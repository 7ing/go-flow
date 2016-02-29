# go-flow

## A cancellable concurrent pattern for Go programming language

Go routine and channels facilitate developers to do concurrent programming. However, it is not easy for a beginner to
write bug-free go-routines. Especially dealing with a complex flow net, make it cancellable is not that straightforward. 

**There are 5 ways to end a go routine:**

1. Successful return void or result(s)
2. Expected error return
3. Unexpected panic / error
4. Job is timeout
5. Job is cancelled

**There are 2 actions to deal with panic:**

1. quit the whole process whenever there is a panic
2. only cancel the problematic go-routine branch (includes its sub-go-routines)

### This package is aim to abstract the flow net, and give a panic-free solution.

***

## GOAL
- User define the flow net
- It completes the whole task or fail in all
- User should get notify whether there is a panic, or error, or job succeed
- User could define the timeout for the whole task

***

## How-To

> Note: this package requires **"golang.org/x/net/context"** package

1st, define a flow net, together with a timeout duration

```
flow := NewFlowNet(1 * time.Millisecond)
```

2nd, define a start node (must-have), a sink node (must-have), and several internal nodes (optional)
Note that, all nodes must have a name tag. They are used by flow control. 

```
start := flow.InitStart("Start")
A := flow.InitNode("A")
B := flow.InitNode("B")
C := flow.InitSink("C")
```

3rd, define actions for each node. The function signature is **func() error**. And each node could
have multiple input and output channels, where all channels' signature is **chan interface{}**. User
could identify input channel by using **node.From(nodeName)** function, and output channel by using 
**node.To(nodeName)**

```
start.Tk = func() error {
		start.To("A") <- 1
		start.To("B") <- "2"
		return nil
}
A.Tk = func() error {
		a := <-A.From("Start")
		A.To("C") <- a
		return nil
}
B.Tk = func() error {
		bStr := <-B.From("Start")
		switch bStr := bStr.(type) { 
		case string:
			B.To("C") <- b
		}
		return nil
}
C.Tk = func() error {
		a := <-C.From("A")
		b := <-C.From("B")
		// do something with a and b
		C.ToSink() <- true // indicate job is done
		return nil
}
```

4th, connect the dots

```
flow.Connect(start, A)
flow.Connect(start, B)
flow.Connect(A, C)
flow.Connect(B, C)
```

5th, run the flow

```
flow.Run()
```

6th, cleanup the flow after use it (optional)

```
flow.Cleanup()
```
