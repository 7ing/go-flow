package flow

import "fmt"

const done string = "SUPER_SINK_NODE"

// Each node can have multiple input and output ports.
// They are connected via channels with interface{} type.
type Ports struct {
	In  map[string]<-chan interface{} // channel name -> read-only channel
	Out map[string]chan<- interface{} // channel name -> write-only channel
}

// Each node in the flow must have an unique name. It is used
// for labeled the connections / edges. The main task is defined
// in Tk function, which take Ps ports as input and output.
// In order to better control the task, user should pass in the
// flow's background.
type Node struct {
	Name string       // node name, cannot be empty
	Ps   *Ports       // node's interfaces
	Tk   func() error // node's job
	Bg   *Background  // background, cannot be nil
}

// Return a new node with specified unique name, which should not be
// nil or empty string. It will help user to initialize ports too.
// Note that, 1) user should define its background and function task before
// using it; 2) "SUPER_SINK_NODE" is a special reserved name.
func NewNode(name string) *Node {
	if len(name) == 0 || name == done {
		panic(fmt.Sprintf("node error : invalid node name"))
	}
	return &Node{
		Name: name,
		Ps: &Ports{
			In:  make(map[string]<-chan interface{}),
			Out: make(map[string]chan<- interface{}),
		},
	}
}

// Run the task defined in Node.Tk function. Normally, it should be
// called by flow.Run(), if this node is joined in the flow.
func (n *Node) Run() {
	done := make(chan bool)
	defer close(done)

	go n.runTask(done) // run node's task in a separate go-routine

	select {
	case <-done: // finished the task
		return
	case <-n.Bg.Ctx.Done(): // timeout or cancelled by background
		return // log.Printf("background exit %s : %s", n.Name, n.Bg.Ctx.Err())
	}
}

// A helper function to run the actual node's task.
// Do not exposed to the caller.
func (n *Node) runTask(done chan<- bool) {
	defer func() /* deal with unexpected panic situation */ {
		if r := recover(); r != nil {
			n.Bg.Err <- fmt.Errorf("panic in %s : %s", n.Name, r) // report the error to background
			return                                                // do not recover from it, exit directly
		}
	}()

	// n.Tk could be a block call, it will exit when all ports are closed
	if err := n.Tk(); err != nil {
		n.Bg.Err <- err // report the error to background
	}

	done <- true // tell node the task is completed
}

// Return the output channel by its receiver node's name
func (n *Node) To(receiver string) chan<- interface{} {
	cName := n.Name + "2" + receiver
	if _, ok := n.Ps.Out[cName]; ok {
		return n.Ps.Out[cName]
	} else {
		panic(fmt.Sprintf("flow error : %s is not connected with %s", receiver, n.Name))
	}
}

// Return the input channel by its sender node's name
func (n *Node) From(sender string) <-chan interface{} {
	cName := sender + "2" + n.Name
	if _, ok := n.Ps.In[cName]; ok {
		return n.Ps.In[sender+"2"+n.Name]
	} else {
		panic(fmt.Sprintf("flow error : %s is not connected with %s", sender, n.Name))
	}
}

// Return a channel for super sink node
func (n *Node) ToSink() chan<- interface{} {
	if _, ok := n.Ps.Out[done]; ok {
		return n.Ps.Out[done]
	} else {
		panic(fmt.Sprintf("flow error : %s is not a sink node", n.Name))
	}
}