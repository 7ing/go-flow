package flow

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

//  Start => (A, B) => C => Done
func TestFlowNet(t *testing.T) {
	flow := NewFlow(1 * time.Millisecond)
	start := flow.InitStart("Start")
	start.Tk = func() error {
		start.To("A") <- 1
		close(start.To("A"))
		start.To("B") <- "2"
		close(start.To("B"))
		fmt.Printf("Start sends : 1 and \"2\"\n")
		return nil
	}

	A := flow.InitNode("A")
	A.Tk = func() error {
		a := <-A.From("Start")
		fmt.Printf("A got : %v\n", a)
		A.To("C") <- a
		return nil
	}

	B := flow.InitNode("B")
	B.Tk = func() error {
		bStr := <-B.From("Start")
		switch bStr := bStr.(type) {
		case string:
			fmt.Printf("B got : \"%s\"\n", bStr)
			b, _ := strconv.Atoi(bStr)
			B.To("C") <- b
		}
		return nil
	}

	C := flow.InitSink("C")
	C.Tk = func() error {
		a := <-C.From("A")
		b := <-C.From("B")
		switch a := a.(type) {
		case int:
			switch b := b.(type) {
			case int:
				fmt.Printf("Sink got sum : %v\n", a+b)
			}
		}
		C.ToSink() <- true
		return nil
	}

	flow.Connect(start, A)
	flow.Connect(start, B)
	flow.Connect(A, C)
	flow.Connect(B, C)

	if err := flow.Run(); err != nil {
		t.Errorf("%s", err)
	}

	flow.Cleanup()
}