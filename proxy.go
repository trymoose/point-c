package wgcaddy

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/trymoose/point-c/wgapi"
	"github.com/trymoose/point-c/wgcaddy/internal/configvalues"
	"net"
)

func init() {
	caddy.RegisterModule(new(Proxy))
}

var (
	_ caddy.Provisioner  = (*Proxy)(nil)
	_ caddy.CleanerUpper = (*Proxy)(nil)
	_ caddy.Module       = (*Proxy)(nil)
	_ caddy.App          = (*Proxy)(nil)
)

type (
	// Network is a map of a unique name to a [Net] tunnel.
	// Since ip addresses may be arbitrary depending on what the application is doing in the tunnel, names are used as lookup.
	// This allows helps with configuration, so that users don't need to remember ip addresses.
	Network interface {
		Networks() map[string]Net
	}
	// Net is a peer in a proxy/vpn. If it has a local address [Net.LocalAddress] should return a non-nil value.
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

// Proxy allows usage of vpn and proxy networks through a [net]-ish interface.
type Proxy struct {
	NetworksRaw []json.RawMessage `json:"networks,omitempty" caddy:"namespace=proxy.net inline_key=type"`
	Networks    []Network         `json:"-"`
	Net         map[string]Net
}

type (
	// Port is a value in the uint16 range. 0 may or may nor be valid depending on the context.
	Port = configvalues.CaddyTextUnmarshaler[uint16, configvalues.Unsigned[uint16], *configvalues.Unsigned[uint16]]
	//PortPair = configvalues.CaddyTextUnmarshaler[[2]uint16, configvalues.PortPair, *configvalues.PortPair]

	// UDPAddr is a wrapper for the [net.UDPAddr] type.
	UDPAddr = configvalues.CaddyTextUnmarshaler[*net.UDPAddr, configvalues.UDPAddr, *configvalues.UDPAddr]
	// IP is wrapper for the [net.IP] type.
	IP = configvalues.CaddyTextUnmarshaler[net.IP, configvalues.IP, *configvalues.IP]

	// Hostname is a unique hostname.
	Hostname = configvalues.CaddyTextUnmarshaler[string, configvalues.String, *configvalues.String]

	// PrivateKey is a wireguard private key in base64 format.
	PrivateKey = configvalues.CaddyTextUnmarshaler[wgapi.PrivateKey, configvalues.WGKey[wgapi.PrivateKey], *configvalues.WGKey[wgapi.PrivateKey]]
	// PublicKey is a wireguard public key in base64 format.
	PublicKey = configvalues.CaddyTextUnmarshaler[wgapi.PublicKey, configvalues.WGKey[wgapi.PublicKey], *configvalues.WGKey[wgapi.PublicKey]]
	// PresharedKey is a wireguard preshared key in base64 format.
	PresharedKey = configvalues.CaddyTextUnmarshaler[wgapi.PresharedKey, configvalues.WGKey[wgapi.PresharedKey], *configvalues.WGKey[wgapi.PresharedKey]]
)

func (*Proxy) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "proxy",
		New: func() caddy.Module { return new(Proxy) },
	}
}

func (wg *Proxy) Provision(ctx caddy.Context) error {
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

func (wg *Proxy) Cleanup() error {
	wg.Networks = nil
	wg.Net = nil
	return nil
}

func (*Proxy) Start() error { return nil }
func (*Proxy) Stop() error  { return nil }

// Lookup gets a [Net] by its declared name.
func (wg *Proxy) Lookup(name string) (Net, bool) {
	n, ok := wg.Net[name]
	return n, ok
}
