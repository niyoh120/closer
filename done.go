package closer

import "sync"

type done struct {
	c    chan struct{}
	once sync.Once
}

func (d *done) close() {
	d.once.Do(func() {
		close(d.c)
	})
}

func newDone() *done {
	d := &done{}
	d.c = make(chan struct{})
	return d
}
