package wgcaddy

import (
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"net"
)

var (
	_ caddy.Provisioner = (*ProxyListener)(nil)
	_ net.Listener      = (*ProxyListener)(nil)
	_ caddy.Module      = (*ProxyListener)(nil)
)

func init() {
	caddy.RegisterModule(new(ProxyListener))
}

type ProxyListener struct {
	Name Hostname
	Port Port
	ln   net.Listener
}

func (p *ProxyListener) Provision(ctx caddy.Context) error {
	m, err := ctx.App("proxy")
	if err != nil {
		return err
	}
	n, ok := m.(*Proxy).Lookup(p.Name.Value())
	if !ok {
		return fmt.Errorf("proxy net %q does not exist", p.Name.Value())
	}

	ln, err := n.Listen(&net.TCPAddr{IP: n.LocalAddr(), Port: int(p.Port.Value())})
	if err != nil {
		return err
	}
	p.ln = ln
	return nil
}

func (p *ProxyListener) Accept() (net.Conn, error) { return p.ln.Accept() }
func (p *ProxyListener) Close() error              { return p.ln.Close() }
func (p *ProxyListener) Addr() net.Addr            { return p.ln.Addr() }

func (*ProxyListener) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "caddy.listeners.merge.listeners.proxy",
		New: func() caddy.Module { return new(ProxyListener) },
	}
}
