package flow

import "fmt"

const DONE string = "Done"

type Ports struct {
	In  map[string]<-chan interface{} // channel name -> read-only channel
	Out map[string]chan<- interface{} // channel name -> write-only channel
}

type Node struct {
	Name string       // node name, cannot be empty
	Ps   *Ports       // node's interfaces
	Tk   func() error // node's job
	Bg   *Background  // background, cannot be nil
}

func NewNode(name string) *Node {
	return &Node{
		Name: name,
		Ps: &Ports{
			In:  make(map[string]<-chan interface{}),
			Out: make(map[string]chan<- interface{}),
		},
	}
}

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

func (n *Node) To(receiver string) chan<- interface{} {
	cName := n.Name + "2" + receiver
	if _, ok := n.Ps.Out[cName]; ok {
		return n.Ps.Out[cName]
	} else {
		panic(fmt.Sprintf("flow error : %s is not connected with %s", receiver, n.Name))
	}
}

func (n *Node) From(sender string) <-chan interface{} {
	cName := sender + "2" + n.Name
	if _, ok := n.Ps.In[cName]; ok {
		return n.Ps.In[sender+"2"+n.Name]
	} else {
		panic(fmt.Sprintf("flow error : %s is not connected with %s", sender, n.Name))
	}
}

func (n *Node) ToSink() chan<- interface{} {
	if _, ok := n.Ps.Out[DONE]; ok {
		return n.Ps.Out[DONE]
	} else {
		panic(fmt.Sprintf("flow error : %s is not a sink node", n.Name))
	}
}