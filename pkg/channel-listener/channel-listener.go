package channel_listener

import (
	"net"
	"sync/atomic"
)

type Listener struct {
	c        chan net.Conn
	done     chan struct{}
	closeErr atomic.Pointer[error]
	addr     net.Addr
}

// New is a listener that passes connections from a channel to the accept method.
func New(in <-chan net.Conn, addr net.Addr) *Listener {
	cl := &Listener{
		c:    make(chan net.Conn),
		done: make(chan struct{}),
		addr: addr,
	}
	go func() {
		for {
			select {
			case <-cl.done:
				return
			case c, ok := <-in:
				if !ok {
					cl.Close()
					return
				}
				select {
				case <-cl.done:
					return
				case cl.c <- c:
				}
			}
		}
	}()
	return cl
}

func (d *Listener) Accept() (net.Conn, error) {
	select {
	case <-d.done:
		return nil, *d.closeErr.Load()
	case c := <-d.c:
		return c, nil
	}
}

func (d *Listener) Close() error { return d.CloseWithErr(net.ErrClosed) }
func (d *Listener) CloseWithErr(err error) error {
	if d.closeErr.CompareAndSwap(nil, &err) {
		close(d.done)
	}
	return nil
}
func (d *Listener) Addr() net.Addr        { return d.addr }
func (d *Listener) Done() <-chan struct{} { return d.done }
