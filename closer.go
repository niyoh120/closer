package closer

import (
	"context"
	"sync"
)

type Closer interface {
	// Close close the closer, return false if the closer already closed.
	Close() bool

	// Closed check weather then closer has been close.
	Closed() bool

	// Waiting all callback and context with the closer done.
	Wait()

	// Chan returns a channel that's closed when the closer closed.
	Chan() <-chan struct{}

	// Context return a cancelable context that's canceled when the closer closed.
	Context() context.Context

	// AddCallbacks add some callback function to the closer which will run in order of addition when the close closed.
	AddCallbacks(...func())

	// WithContext associate the closer with a context, the closer closed when the context's done channel closed.
	// Call this method on a closed closer do nothing.
	WithContext(context.Context)
}

// New return a closer.
func New() Closer {
	return newCloser()
}

// WithContext return a closer associate with a context.
func WithContext(ctx context.Context) Closer {
	c := newCloser()
	c.WithContext(ctx)
	return c
}

type closer struct {
	locker    sync.Mutex
	closed    bool
	done      *done
	wg        sync.WaitGroup
	c         chan struct{}
	callbacks []func()
	ctx       context.Context
	cancel    context.CancelFunc
}

func (c *closer) Close() bool {
	defer c.locker.Unlock()
	c.locker.Lock()
	c.done.close()
	if !c.closed {
		c.closed = true
		return true
	}
	return false
}

func (c *closer) Closed() bool {
	defer c.locker.Unlock()
	c.locker.Lock()
	return c.closed
}

func (c *closer) Wait() {
	<-c.c
}

func (c *closer) Chan() <-chan struct{} {
	return c.ctx.Done()
}

func (c *closer) Context() context.Context {
	return c.ctx
}

func (c *closer) AddCallbacks(cbs ...func()) {
	add := func() bool {
		defer c.locker.Unlock()
		c.locker.Lock()
		if !c.closed {
			c.callbacks = append(c.callbacks, cbs...)
		}
		return c.closed
	}

	call := func() {
		for _, cb := range cbs {
			cb()
		}
	}

	if add() {
		go call()
	}
}

func (c *closer) WithContext(ctx context.Context) {
	defer c.locker.Unlock()
	c.locker.Lock()
	if c.closed {
		return
	}

	c.resetDone()

	doneC := c.done
	go func() {
		select {
		case <-doneC.c:
			return
		case <-ctx.Done():
			c.Close()
		}
	}()
}

func (c *closer) init() {
	c.locker.Lock()
	doneC := c.done
	c.locker.Unlock()

	c.wg.Add(1)
	go func() {
		<-doneC.c
		c.wg.Done()
	}()

	go func() {
		c.wg.Wait()

		c.locker.Lock()
		c.cancel()
		callbacks := make([]func(), len(c.callbacks))
		copy(callbacks, c.callbacks)
		c.callbacks = nil
		c.locker.Unlock()

		for _, cb := range callbacks {
			cb()
		}
		<-c.ctx.Done()
		close(c.c)
	}()
}

func (c *closer) resetDone() {
	c.wg.Add(1)
	c.done.close()
	c.done = newDone()

	go func() {
		<-c.done.c
		c.wg.Done()
	}()
}

func newCloser() *closer {
	c := &closer{}
	c.callbacks = make([]func(), 0)
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.done = newDone()
	c.c = make(chan struct{})
	c.init()

	return c
}
