package closer

import (
	"context"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func TestClose(t *testing.T) {
	ass := assert.New(t)

	c := New()
	ass.Equal(false, c.Closed())
	ass.True(c.Close())
	c.Wait()
	ass.Equal(true, c.Closed())
	ass.False(c.Close())
}

func TestChan(t *testing.T) {
	ass := assert.New(t)

	flag := false
	c := New()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-c.Chan()
		ass.Equal(flag, true)
	}()

	flag = true
	c.Close()
	wg.Wait()
}

func TestAddCallbacks(t *testing.T) {
	ass := assert.New(t)

	c := New()
	rand.Seed(time.Now().Unix())
	g := func(c1, c2 chan<- int) func() {
		return func() {
			n := rand.Int()
			c1 <- n
			c2 <- n
		}
	}

	c1 := make(chan int, 10)
	c2 := make(chan int, 10)

	for i := 0; i < 5; i++ {
		c.AddCallbacks(g(c1, c2), g(c1, c2))
	}
	c.Close()
	for i := 0; i < 10; i++ {
		ass.Equal(<-c1, <-c2)
	}

	c.AddCallbacks(g(c1, c2))
	ass.Equal(<-c1, <-c2)
}

func TestContext(t *testing.T) {
	ass := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	c1 := WithContext(ctx)
	c2 := WithContext(c1.Context())
	cancel()
	c1.Wait()
	c2.Wait()
	ass.Equal(true, c1.Closed())
	ass.Equal(true, c2.Closed())
}
