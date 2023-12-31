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
	_ caddy.Provisioner     = (*MultiWrapper)(nil)
	_ caddy.CleanerUpper    = (*MultiWrapper)(nil)
	_ caddy.ListenerWrapper = (*MultiWrapper)(nil)
	_ caddy.Module          = (*MultiWrapper)(nil)
	_ caddyfile.Unmarshaler = (*MultiWrapper)(nil)
)

// MultiWrapper loads multiple [net.Listener]s and aggregates their [net.Conn]s into a single [net.Listener].
// It allows caddy to accept connections from multiple sources.
type MultiWrapper struct {
	// ListenerRaw is a slice of JSON-encoded data representing listener configurations.
	// These configurations are used to create the actual net.Listener instances.
	// Listeners should implement [net.Listener] and be in the 'caddy.listeners.multi.listeners' namespace.
	ListenerRaw []json.RawMessage `json:"listeners" caddy:"namespace=caddy.listeners.multi.listeners inline_key=listener"`

	// listeners is a slice of net.Listener instances created based on the configurations
	// provided in ListenerRaw. These listeners are the actual network listeners that
	// will be accepting connections.
	listeners []net.Listener

	// conns is a channel for net.Conn instances. Connections accepted by any of the
	// listeners in the 'listeners' slice are sent through this channel.
	// This channel is passed to the constructor of [channel_listener.Listener].
	conns chan net.Conn
}

// CaddyModule implements [caddy.Module].
func (p *MultiWrapper) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "caddy.listeners.multi",
		New: func() caddy.Module { return new(MultiWrapper) },
	}
}

// Provision implements [caddy.Provisioner].
// It loads the listeners from their configs and asserts them to [net.Listener].
// Any failed assertions will cause a panic.
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

// Cleanup implements [caddy.CleanerUpper].
// All wrapped listeners are closed and the struct is cleared.
func (p *MultiWrapper) Cleanup() (err error) {
	for len(p.listeners) > 0 {
		err = errors.Join(err, p.listeners[0].Close())
		p.listeners = p.listeners[1:]
	}
	*p = MultiWrapper{}
	return
}

// WrapListener implements [caddy.ListenerWrapper].
// The listener passed in is closed by [MultiWrapper] during cleanup.
func (p *MultiWrapper) WrapListener(ls net.Listener) net.Listener {
	p.listeners = append(p.listeners, ls)
	cl := channel_listener.New(p.conns, ls.Addr())
	for _, ls := range p.listeners {
		go listen(ls, p.conns, cl.Done(), cl.CloseWithErr)
	}
	return cl
}

// listen manages incoming network connections on a given listener.
// It sends accepted connections to the 'conns' channel. When a
// signal is sent to the 'done' channel any accepted connections not passed on are closed and ignored.
// In case of an error during accepting a connection, it calls the 'finish' function with the error.
func listen(ls net.Listener, conns chan<- net.Conn, done <-chan struct{}, finish func(error) error) {
	for {
		c, err := ls.Accept()
		if err != nil {
			// If one connection errors on Accept, pass the error on and close all other connections.
			// Only the first error from an Accept will be passed on.
			finish(err)
			return
		}

		select {
		case <-done:
			// The connection has been closed, close the received connection and ignore it.
			c.Close()
			continue
		case conns <- c:
			// Connection has been accepted
		}
	}
}

// UnmarshalCaddyfile implements [caddyfile.Unmarshaler].
// Must have at least one listener to aggregate with the wrapped listener.
// `tls` should come specifically after any `multi` directives.
//
//	 http caddyfile:
//		{
//		  servers :443 {
//		    listener_wrappers {
//		      multi {
//		        <submodule name> <submodule config>
//		      }
//		      tls
//		    }
//		  }
//		}
func (p *MultiWrapper) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for nesting := d.Nesting(); d.NextBlock(nesting); {
			modName := d.Val()
			v, err := caddyfile.UnmarshalModule(d, "caddy.listeners.multi.listeners."+modName)
			if err != nil {
				return err
			}

			p.ListenerRaw = append(p.ListenerRaw, caddyconfig.JSON(v, nil))
		}
	}
	return nil
}
