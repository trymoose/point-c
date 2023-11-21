package point_c

import (
	"context"
	"net"
	"sync/atomic"
)

func init() {
	caddy.RegisterNetwork("dummy", func(_ context.Context, _, addr string, _ net.ListenConfig) (any, error) {
		return NewChannelListener(make(<-chan net.Conn), dummyAddr(addr)), nil
	})
}

type dummyAddr string

func (d dummyAddr) Network() string { return "dummy" }
func (d dummyAddr) String() string  { return string(d) }

type ChannelListener struct {
	c        chan net.Conn
	done     chan struct{}
	closeErr atomic.Pointer[error]
	addr     net.Addr
}

// NewChannelListener is a listener that passes connections from a channel to the accept method.
func NewChannelListener(in <-chan net.Conn, addr net.Addr) *ChannelListener {
	cl := &ChannelListener{
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

func (d *ChannelListener) Accept() (net.Conn, error) {
	select {
	case <-d.done:
		return nil, *d.closeErr.Load()
	case c := <-d.c:
		return c, nil
	}
}

func (d *ChannelListener) Close() error { return d.CloseWithErr(net.ErrClosed) }
func (d *ChannelListener) CloseWithErr(err error) error {
	if d.closeErr.CompareAndSwap(nil, &err) {
		close(d.done)
	}
	return nil
}
func (d *ChannelListener) Addr() net.Addr        { return d.addr }
func (d *ChannelListener) Done() <-chan struct{} { return d.done }
