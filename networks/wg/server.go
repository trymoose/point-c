package wg

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	pointc "github.com/trymoose/point-c"
	"github.com/trymoose/point-c/pkg/configvalues"
	"github.com/trymoose/point-c/pkg/wg"
	"github.com/trymoose/point-c/pkg/wg/wgapi/wgconfig"
	"github.com/trymoose/point-c/pkg/wg/wglog/wgevents"
	"go.mrchanchal.com/zaphandler"
	"log/slog"
	"maps"
	"net"
)

var (
	_ caddy.Module       = (*Server)(nil)
	_ caddy.Provisioner  = (*Server)(nil)
	_ caddy.CleanerUpper = (*Server)(nil)
	_ pointc.Network     = (*Server)(nil)
	_ json.Marshaler     = (*Server)(nil)
	_ json.Unmarshaler   = (*Server)(nil)
)

func init() {
	caddy.RegisterModule(new(Server))
}

func (*Server) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "proxy.net.wireguard-server",
		New: func() caddy.Module { return new(Server) },
	}
}

// Server is a basic wireguard server.
type Server struct {
	json struct {
		Name       configvalues.Hostname
		IP         configvalues.IP
		ListenPort configvalues.Port
		Private    PrivateKey
		Peers      []struct {
			Name         configvalues.Hostname
			Public       PublicKey
			PresharedKey PresharedKey
			IP           configvalues.IP
		}
	}
	net    *wg.Net
	logger *slog.Logger
	wg     *wg.Wireguard
	nets   map[string]pointc.Net
}

func (c *Server) UnmarshalJSON(bytes []byte) error { return json.Unmarshal(bytes, &c.json) }
func (c *Server) MarshalJSON() ([]byte, error)     { return json.Marshal(c.json) }

func (c *Server) Networks() map[string]pointc.Net { return maps.Clone(c.nets) }

func (c *Server) Cleanup() error { return c.wg.Close() }

func (c *Server) Provision(ctx caddy.Context) (err error) {
	*c = Server{
		json:   c.json,
		logger: slog.New(zaphandler.New(ctx.Logger())),
		nets:   map[string]pointc.Net{},
	}
	c.nets[c.json.Name.Value()] = &serverNet{srv: c, ip: c.json.IP.Value()}

	cfg := wgconfig.Server{
		Private:    c.json.Private.Value(),
		ListenPort: c.json.ListenPort.Value(),
	}
	for _, peer := range c.json.Peers {
		cfg.AddPeer(peer.Public.Value(), peer.PresharedKey.Value(), peer.IP.Value())
		if _, ok := c.nets[peer.Name.Value()]; ok {
			return fmt.Errorf("hostname %q already declared in config", peer.Name.Value())
		}
		c.nets[c.json.Name.Value()] = &serverNet{srv: c, ip: peer.IP.Value()}
	}

	c.wg, err = wg.New(
		wg.OptionConfig(&cfg),
		wg.OptionLogger(wgevents.Events(func(e wgevents.Event) { e.Slog(c.logger) })),
		wg.OptionNetDevice(&c.net),
	)
	return
}

var (
	_ pointc.Net    = (*serverNet)(nil)
	_ pointc.Dialer = (*serverDialer)(nil)
)

type (
	serverNet struct {
		srv *Server
		ip  net.IP
	}
	serverDialer struct {
		d *wg.Dialer
	}
)

func (s *serverNet) Listen(addr *net.TCPAddr) (net.Listener, error) { return s.srv.net.Listen(addr) }

func (s *serverNet) ListenPacket(addr *net.UDPAddr) (net.PacketConn, error) {
	return s.srv.net.ListenPacket(addr)
}

func (s *serverNet) Dialer(laddr net.IP, port uint16) pointc.Dialer {
	return &serverDialer{d: s.srv.net.Dialer(laddr, port)}
}

func (s *serverNet) LocalAddr() net.IP { return s.ip }

func (s *serverDialer) Dial(ctx context.Context, addr *net.TCPAddr) (net.Conn, error) {
	return s.d.DialTCP(ctx, addr)
}

func (s *serverDialer) DialPacket(addr *net.UDPAddr) (net.PacketConn, error) {
	return s.d.DialUDP(addr)
}
