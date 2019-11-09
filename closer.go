// Closer interface for clean resource
package closer

import (
	"context"
	"sync"
)

type Closer interface {
	// Close close the closer, return false if the closer already closed.
	Close() bool

	// Closed check whether the closer has been close.
	Closed() bool

	// Waiting all callbacks within the closer done.
	Wait(context.Context) error

	// Chan returns a channel closed when the closer closed.
	Chan() <-chan struct{}

	// Context return a context canceled when the closer closed.
	Context() context.Context

	// AddCallbacks add some callback function to the closer.
	// All callbacks will run in order of addition in a goroutine when the closer closed.
	// Call this method on a closed closer will create a goroutine to run callbacks immediately.
	AddCallbacks(...func())

	// WithContext associate the closer with a context, the closer closed when the context's done channel closed.
	// If the closer already associate with a context, it will be replace by the new one.
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
	locker sync.Mutex
	done   chan struct{}
	ref    uint32

	callbacks []func()

	ctx    context.Context
	cancel context.CancelFunc
}

func newCloser() *closer {
	c := &closer{}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.ref++

	return c
}

func (c *closer) Close() bool {
	defer c.locker.Unlock()
	c.locker.Lock()
	if c.ref == 0 {
		return false
	}
	c.close()
	if c.done != nil {
		close(c.done)
	}
	return true
}

func (c *closer) Closed() bool {
	defer c.locker.Unlock()
	c.locker.Lock()
	return c.ref == 0
}

func (c *closer) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.ctx.Done():
		return nil
	}
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
		closed := c.ref == 0
		if !closed {
			if c.callbacks == nil {
				c.callbacks = make([]func(), 0, len(cbs))
			}
			c.callbacks = append(c.callbacks, cbs...)
		}
		return closed
	}

	if !add() {
		return
	}

	go func() {
		for _, cb := range cbs {
			cb()
		}
	}()
}

func (c *closer) WithContext(ctx context.Context) {
	defer c.locker.Unlock()
	c.locker.Lock()
	if c.ref == 0 {
		return
	}

	c.resetDone()

	done := c.done
	go func() {
		select {
		case <-done:
		case <-ctx.Done():
		}
		c.Close()
	}()
}

func (c *closer) close() {
	c.ref--
	if c.ref > 0 {
		return
	}
	callbacks := make([]func(), len(c.callbacks))
	copy(callbacks, c.callbacks)
	c.callbacks = nil

	go func() {
		for _, cb := range callbacks {
			cb()
		}
		c.cancel()
	}()
}

func (c *closer) resetDone() {
	if (c.done != nil) {
		c.ref++
		close(c.done)
	}
	c.done = make(chan struct{})
}
