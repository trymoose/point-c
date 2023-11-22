package point_c

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"net"
)

func init() {
	caddy.RegisterModule(new(Pointc))
	httpcaddyfile.RegisterGlobalOption("point-c", func(d *caddyfile.Dispenser, _ any) (any, error) {
		v, err := caddyfile.UnmarshalModule(d, "point-c")
		if err != nil {
			return nil, err
		}
		return &httpcaddyfile.App{Name: "point-c", Value: caddyconfig.JSON(v, nil)}, nil
	})
}

var (
	_ caddy.Provisioner  = (*Pointc)(nil)
	_ caddy.CleanerUpper = (*Pointc)(nil)
	_ caddy.Module       = (*Pointc)(nil)
	_ caddy.App          = (*Pointc)(nil)
)

type (
	// Network is a map of a unique name to a [Net] tunnel.
	// Since ip addresses may be arbitrary depending on what the application is doing in the tunnel, names are used as lookup.
	// This allows helps with configuration, so that users don't need to remember ip addresses.
	Network interface {
		Networks() map[string]Net
	}
	// Net is a peer in the networking stack. If it has a local address [Net.LocalAddress] should return a non-nil value.
	Net interface {
		Listen(addr *net.TCPAddr) (net.Listener, error)
		ListenPacket(addr *net.UDPAddr) (net.PacketConn, error)
		// Dialer returns a [Dialer] with a given local address. If the network does not support arbitrary remote addresses this value can be ignored.
		Dialer(laddr net.IP, port uint16) Dialer
		LocalAddr() net.IP
	}
	Dialer interface {
		Dial(context.Context, *net.TCPAddr) (net.Conn, error)
		DialPacket(*net.UDPAddr) (net.PacketConn, error)
	}
)

// Pointc allows usage of networks through a [net]-ish interface.
type Pointc struct {
	NetworksRaw []json.RawMessage `json:"networks,omitempty" caddy:"namespace=point-c.net inline_key=type"`
	Networks    []Network         `json:"-"`
	Net         map[string]Net    `json:"-"`
}

func (*Pointc) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "point-c",
		New: func() caddy.Module { return new(Pointc) },
	}
}

func (wg *Pointc) Provision(ctx caddy.Context) error {
	if wg.NetworksRaw != nil {
		val, err := ctx.LoadModule(wg, "NetworksRaw")
		if err != nil {
			return fmt.Errorf("failed to load wireguard networks: %w", err)
		}
		raw, ok := val.([]any)
		if !ok {
			return fmt.Errorf("invalid raw module slice %T", val)
		}

		wg.Networks = make([]Network, len(wg.NetworksRaw))
		for i, v := range raw {
			wg.Networks[i] = v.(Network)
		}

		wg.Net = map[string]Net{}
		for _, n := range wg.Networks {
			for name, nn := range n.Networks() {
				if _, ok := wg.Net[name]; ok {
					return fmt.Errorf("net %q declared twice", name)
				}
				wg.Net[name] = nn
			}
		}
	}
	return nil
}

func (wg *Pointc) Cleanup() error {
	wg.Networks = nil
	wg.Net = nil
	return nil
}

func (*Pointc) Start() error { return nil }
func (*Pointc) Stop() error  { return nil }

// Lookup gets a [Net] by its declared name.
func (wg *Pointc) Lookup(name string) (Net, bool) {
	n, ok := wg.Net[name]
	return n, ok
}

func (wg *Pointc) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		for nesting := d.Nesting(); d.NextBlock(nesting); {
			modName := d.Val()
			if modName == "" {
				continue
			}

			v, err := caddyfile.UnmarshalModule(d, "point-c.net."+modName)
			if err != nil {
				return err
			}

			raw, err := json.Marshal(v)
			if err != nil {
				return err
			}
			wg.NetworksRaw = append(wg.NetworksRaw, raw)
		}
	}
	return nil
}
