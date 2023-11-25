package point_c

import (
	"encoding/json"
	"errors"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/trymoose/point-c/pkg/channel-listener"
	"net"
)

var (
	_ caddy.Provisioner     = (*MergeWrapper)(nil)
	_ caddy.CleanerUpper    = (*MergeWrapper)(nil)
	_ caddy.ListenerWrapper = (*MergeWrapper)(nil)
	_ caddy.Module          = (*MergeWrapper)(nil)
	_ caddyfile.Unmarshaler = (*MergeWrapper)(nil)
)

// MergeWrapper wraps the base connection with multiple wrappers. The returned wrapper produces connections from all [net.Listener]s given.
// Listeners merged together will be owned by this module. When the module's [caddy.CleanerUpper] is called the listeners will be closed.
type MergeWrapper struct {
	ListenerRaw []json.RawMessage `json:"listeners" caddy:"namespace=caddy.listeners.merge.listeners inline_key=listener"`
	listeners   []net.Listener
	conns       chan net.Conn
}

func (p *MergeWrapper) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "caddy.listeners.merge",
		New: func() caddy.Module { return new(MergeWrapper) },
	}
}

func (p *MergeWrapper) Provision(ctx caddy.Context) error {
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

func (p *MergeWrapper) Cleanup() (err error) {
	for len(p.listeners) > 0 {
		err = errors.Join(err, p.listeners[0].Close())
		p.listeners = p.listeners[1:]
	}
	*p = MergeWrapper{}
	return
}

func (p *MergeWrapper) WrapListener(ls net.Listener) net.Listener {
	p.listeners = append(p.listeners, ls)
	cl := channel_listener.New(p.conns, ls.Addr())
	for _, ls := range p.listeners {
		go listen(ls, p.conns, cl.Done(), cl.CloseWithErr)
	}
	return cl
}

// listen does the actual listening.
func listen(ls net.Listener, conns chan<- net.Conn, done <-chan struct{}, finish func(error) error) {
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
		case conns <- c:
		}
	}
}

// UnmarshalCaddyfile unmarshals the caddyfile.
//
//	{
//	  servers :443 {
//	    listener_wrappers {
//	      merge {
//	        <submodule name> <submodule config>
//	      }
//	      tls
//	    }
//	  }
//	}
func (p *MergeWrapper) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for nesting := d.Nesting(); d.NextBlock(nesting); {
			modName := d.Val()
			v, err := caddyfile.UnmarshalModule(d, "caddy.listeners.merge.listeners."+modName)
			if err != nil {
				return err
			}

			p.ListenerRaw = append(p.ListenerRaw, caddyconfig.JSON(v, nil))
		}
	}
	return nil
}
