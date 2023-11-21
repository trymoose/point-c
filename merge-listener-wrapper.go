package point_c

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
)

var (
	_ caddy.Provisioner     = (*MultiWrapper)(nil)
	_ caddy.CleanerUpper    = (*MultiWrapper)(nil)
	_ caddy.ListenerWrapper = (*MultiWrapper)(nil)
	_ caddy.Module          = (*MultiWrapper)(nil)
)

// MultiWrapper wraps the base connection with multiple wrappers. The returned wrapper produces connections from all [net.Listener]s given.
// Listeners merged together will be owned by this module. When the module's [caddy.CleanerUpper] is called the listeners will be closed.
type MultiWrapper struct {
	ListenerRaw []json.RawMessage `json:"listener" caddy:"namespace=caddy.listeners.merge.listeners inline_key=listener"`
	listeners   []net.Listener
	logger      *slog.Logger
	conns       chan net.Conn
}

func (p *MultiWrapper) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "caddy.listeners.merge",
		New: func() caddy.Module { return new(MultiWrapper) },
	}
}

func (p *MultiWrapper) Provision(ctx caddy.Context) error {
	if p.ListenerRaw == nil {
		return errors.New("no listener provided")
	}
	v, err := ctx.LoadModule(p, "ListenerRaw")
	if err != nil {
		return err
	}

	p.listeners = make([]net.Listener, len(v.([]any)))
	for i, ls := range v.([]any) {
		p.listeners[i] = ls.(net.Listener)
	}

	p.conns = make(chan net.Conn)
	return nil
}

func (p *MultiWrapper) Cleanup() (err error) {
	for len(p.listeners) > 0 {
		err = errors.Join(err, p.listeners[0].Close())
		p.listeners = p.listeners[1:]
	}
	*p = MultiWrapper{}
	return
}

func (p *MultiWrapper) WrapListener(ls net.Listener) net.Listener {
	p.listeners = append(p.listeners, ls)
	cl := NewChannelListener(p.conns, ls.Addr())
	for _, ls := range p.listeners {
		p.listen(ls, cl.Done(), cl.CloseWithErr)
	}
	return cl
}

func (p *MultiWrapper) listen(ls net.Listener, done <-chan struct{}, finish func(error) error) {
	go func() {
		for {
			c, err := ls.Accept()
			if err != nil {
				finish(err)
				return
			}

			select {
			case <-done:
				c.Close()
				return
			case p.conns <- c:
			}
		}
	}()
}
