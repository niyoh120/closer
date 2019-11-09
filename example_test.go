package closer

import (
	"context"
	"fmt"
)

func Example() {
	c := New()
	cb1 := func() {
		fmt.Print("hello ")
	}
	cb2 := func() {
		fmt.Println("world")
	}
	c.AddCallbacks(cb1, cb2)
	c.Close()
	_ = c.Wait(context.Background())

	//Output: hello world
}
