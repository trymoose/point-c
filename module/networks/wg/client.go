package wg

import (
	"context"
	"encoding/json"
	"github.com/caddyserver/caddy/v2"
	pointc "github.com/trymoose/point-c"
	"github.com/trymoose/point-c/pkg/configvalues"
	"github.com/trymoose/point-c/pkg/wg"
	"github.com/trymoose/point-c/pkg/wg/wgapi/wgconfig"
	"github.com/trymoose/point-c/wglog/wgevents"
	"go.mrchanchal.com/zaphandler"
	"log/slog"
	"net"
)

var (
	_ caddy.Module       = (*Client)(nil)
	_ caddy.Provisioner  = (*Client)(nil)
	_ caddy.CleanerUpper = (*Client)(nil)
	_ pointc.Network     = (*Client)(nil)
	_ json.Marshaler     = (*Client)(nil)
	_ json.Unmarshaler   = (*Client)(nil)
)

func init() {
	caddy.RegisterModule(new(Client))
}

func (*Client) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "proxy.net.wireguard-client",
		New: func() caddy.Module { return new(Client) },
	}
}

// Client is a basic wireguard client.
type Client struct {
	json struct {
		Name      configvalues.Hostname
		Endpoint  configvalues.UDPAddr
		IP        configvalues.IP
		Private   PrivateKey
		Public    PublicKey
		Preshared PresharedKey
	}
	name   string
	ip     net.IP
	net    *wg.Net
	logger *slog.Logger
	wg     *wg.Wireguard
}

func (c *Client) UnmarshalJSON(bytes []byte) error { return json.Unmarshal(bytes, &c.json) }
func (c *Client) MarshalJSON() ([]byte, error)     { return json.Marshal(c.json) }

func (c *Client) Networks() map[string]pointc.Net {
	return map[string]pointc.Net{c.name: (*clientNet)(c)}
}

type (
	clientNet    Client
	clientDialer struct{ d *wg.Dialer }
)

func (c *clientDialer) Dial(ctx context.Context, addr *net.TCPAddr) (net.Conn, error) {
	return c.d.DialTCP(ctx, addr)
}
func (c *clientDialer) DialPacket(addr *net.UDPAddr) (net.PacketConn, error) {
	return c.d.DialUDP(addr)
}
func (c *clientNet) Listen(addr *net.TCPAddr) (net.Listener, error) { return c.net.Listen(addr) }
func (c *clientNet) LocalAddr() net.IP                              { return c.ip }
func (c *clientNet) ListenPacket(addr *net.UDPAddr) (net.PacketConn, error) {
	return c.net.ListenPacket(addr)
}
func (c *clientNet) Dialer(laddr net.IP, port uint16) pointc.Dialer {
	return &clientDialer{d: c.net.Dialer(laddr, port)}
}

func (c *Client) Cleanup() error { return c.wg.Close() }

func (c *Client) Provision(ctx caddy.Context) (err error) {
	*c = Client{
		json:   c.json,
		name:   c.json.Name.Value(),
		ip:     c.json.IP.Value(),
		logger: slog.New(zaphandler.New(ctx.Logger())),
	}

	cfg := wgconfig.Client{
		Private:   c.json.Private.Value(),
		Public:    c.json.Public.Value(),
		PreShared: c.json.Preshared.Value(),
		Endpoint:  *c.json.Endpoint.Value(),
	}
	cfg.DefaultPersistentKeepAlive()
	cfg.AllowAllIPs()
	// Hostname is a unique hostname.
	c.wg, err = wg.New(
		wg.OptionConfig(&cfg),
		wg.OptionLogger(wgevents.Events(func(e wgevents.Event) { e.Slog(c.logger) })),
		wg.OptionNetDevice(&c.net),
	)
	return
}
