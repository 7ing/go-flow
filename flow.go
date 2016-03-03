package flow

import (
	"fmt"
	"time"

	"golang.org/x/net/context"
)

// Flow's background, which defines its timed context
// and a global error channel.
type Background struct {
	Ctx context.Context // background context
	Err chan<- error    // write-only error channel
}

// A flow is defined as:
// A map for all connections / edges; a map for all nodes;
// a super start node, which is regarded as flow's switch;
// a timeout duration, which defines max running time for
// the flow; and a Done channel, which indicate the flow
// is completed or not by using a bool value.
type Flow struct {
	Conn  map[string]chan interface{} // edge name -> channel
	Nodes map[string]*Node            // node name -> node
	Start *Node                       // super start node, not in Nodes map
	TOut  time.Duration               // timeout definition
	Done  chan interface{}            // done channel, named as "Done"
}

// Init a new flow with a timeout value.
func NewFlow(time time.Duration) *Flow {
	return &Flow{
		Conn:  make(map[string]chan interface{}),
		Nodes: make(map[string]*Node),
		Done:  make(chan interface{}),
		TOut:  time,
	}
}

// Init super start node. It will be regarded as flow.Start.
func (fn *Flow) InitStart(name string) *Node {
	node := NewNode(name)
	fn.Start = node
	return node
}

// Init normal node. It will be added to flow.Nodes map
func (fn *Flow) InitNode(name string) *Node {
	node := NewNode(name)
	fn.Nodes[name] = node
	return node
}

// Init sink node. It will be added to flow.Nodes map.
// This is a special node, which has an output pointing to
// flow's Done channel (aka. super sink node).
func (fn *Flow) InitSink(name string) *Node {
	node := NewNode(name)
	node.Ps.Out[done] = fn.Done // sink has a DONE channel
	fn.Nodes[name] = node
	return node
}

// Connect two nodes in the flow. Channel will be named as
// {n1.Name}2{n2.Name} to indicate the direction.
func (fn *Flow) Connect(n1, n2 *Node) {
	edgeName := fmt.Sprintf("%s2%s", n1.Name, n2.Name)
	edge := make(chan interface{})
	fn.Conn[edgeName] = edge
	n1.Ps.Out[edgeName] = edge
	n2.Ps.In[edgeName] = edge
}

// Run the flow. It will return error when task is
// incomplete, or timeout, or an error occur at any flow node.
// Otherwise, it return nil to indicate the flow is done successfully.
func (fn *Flow) Run() error {
	errChan := make(chan error)
	defer close(errChan)
	ctx, cancel := context.WithTimeout(context.Background(), fn.TOut)
	defer cancel()

	bg := &Background{
		Ctx: ctx,
		Err: errChan,
	}

	for _, node := range fn.Nodes /* bring up all nodes except start and sink */ {
		node.Bg = bg
		go node.Run()
	}
	fn.Start.Bg = bg
	go fn.Start.Run() // start the whole flow

	select {
	case ok := <-fn.Done:
		switch ok := ok.(type) {
		case bool:
			if ok {
				return nil
			} else {
				return fmt.Errorf("Task incomplete.")
			}
		default:
			return nil
		}
	case <-ctx.Done():
		return fmt.Errorf("Task incomplete : %s", ctx.Err())
	case e := <-errChan:
		return e
	}
}

// Cleanup all opened channel to save resources. 
func (fn *Flow) Cleanup() {
	defer func() {
		recover() // resume from a panic in case closing a closed channel
	}()
	for _, ch := range fn.Conn {
		close(ch)
	}
	close(fn.Done)
}
