// Package wgapi helps with communicating with the userspace wireguard module.
// Since wireguard-go uses a text based configuration this helps with programmatically creating and reading a config.
// Please consult [wireguard cross-platform documentation] for more information on configuration values.
//
// [wireguard cross-platform documentation]: https://www.wireguard.com/xplatform/
package wgapi

import (
	"bytes"
	"fmt"
	"github.com/trymoose/point-c/pkg/wg/wgapi/internal"
	"github.com/trymoose/point-c/pkg/wg/wgapi/internal/key"
	"github.com/trymoose/point-c/pkg/wg/wgapi/internal/value"
	"github.com/trymoose/point-c/pkg/wg/wgapi/internal/value/wgkey"
	"golang.zx2c4.com/wireguard/ipc"
	"io"
	"net"
)

type (
	// Configurable is something that can be converted into a reader that supplies 'key=value\n' values corresponding to the wireguard userspace configuration [wireguard cross-platform documentation].
	Configurable interface {
		WGConfig() io.Reader
	}

	// IPCKeyValue is string key and value pair. The value is represented by [fmt.Stringer].
	IPCKeyValue = internal.KeyValue
)

// IPC is an IPC operation as documented by [wireguard cross-platform documentation].
type IPC []IPCKeyValue

func (ir IPC) WGConfig() io.Reader {
	var buf bytes.Buffer
	for _, e := range ir {
		buf.WriteString(e.Key())
		buf.WriteRune('=')
		buf.WriteString(e.String())
		buf.WriteRune('\n')
	}

	return &buf
}

type (
	// PrivateKey is a private key usable by the IPC.
	PrivateKey = value.Key[key.PrivateKey, wgkey.Private]
	// PublicKey is a public key usable by the IPC.
	PublicKey = value.Key[key.PublicKey, wgkey.Public]
	// PresharedKey is a preshared key usable by the IPC.
	PresharedKey = value.Key[key.PresharedKey, wgkey.PreShared]
)

// DefaultPersistentKeepalive is the default persistent keepalive interval.
const DefaultPersistentKeepalive PersistentKeepalive = 25

// PersistentKeepalive is the interval to send a persistent keepalive packet. Special value 0 disables this.
type PersistentKeepalive = value.Uint16[key.PersistentKeepalive]

type (
	// ReplacePeers replaces all the peers.
	ReplacePeers = value.True[key.ReplacePeers]
	// Remove removes the peer.
	Remove = value.True[key.Remove]
	// UpdateOnly only updates the peer if it is already present.
	UpdateOnly = value.True[key.UpdateOnly]
	// ReplaceAllowedIPs replaces the current allowed IPs instead of appending.
	ReplaceAllowedIPs = value.True[key.ReplaceAllowedIPs]
	// ProtocolVersion is the version of the protocol. Generally not used.
	ProtocolVersion = value.One[key.ProtocolVersion]
	// Get specifies a get operation. Not used unless communicating with an external endpoint.
	Get = value.One[key.Get]
	// Set specifies a set operation. Not used unless communicating with an external endpoint.
	Set = value.One[key.Set]
)

type (
	// LastHandshakeTimeSec is the seconds since the last handshake relative to unix epoch.
	LastHandshakeTimeSec = value.Int64[key.LastHandshakeTimeSec]
	// LastHandshakeTimeNSec is the nanoseconds resolution of the last handshake relative to unix epoch.
	LastHandshakeTimeNSec = value.Int64[key.LastHandshakeTimeNSec]
)

type (
	// RXBytes are the number of received bytes. Only present in a get operation.
	RXBytes = value.Uint64[key.RXBytes]
	// TXBytes are the number of transferred bytes. Only present in a get operation.
	TXBytes = value.Uint64[key.TXBytes]
)

// DefaultListenPort is the default wireguard server port.
const DefaultListenPort ListenPort = 51820

type (
	// Endpoint is the address:port of a wireguard server
	Endpoint = value.UDPAddr[key.Endpoint]
	// AllowedIP is an address allowed to communicate in the tunnel.
	AllowedIP = value.IPNet[key.AllowedIP]
	// ListenPort is the system port used to listen for wireguard traffic.
	ListenPort = value.Uint16[key.ListenPort]
	// FWMark configures the interface as specified in [wireguard cross-platform documentation]. The special value 0 clears the FWMark.
	FWMark = value.Uint32[key.FWMark]
)

// EmptySubnet is the 0.0.0.0/0 subnet
var EmptySubnet AllowedIP

var _ = func() AllowedIP {
	const allSubnets = "0.0.0.0/0"
	_, ip, err := net.ParseCIDR(allSubnets)
	if err != nil {
		panic(fmt.Errorf("failed to parse %q into %T: %w", allSubnets, ip, err))
	}
	EmptySubnet = AllowedIP(*ip)
	return EmptySubnet
}()

// IdentitySubnet converts an IP (v4 or v6) to ipv6/128.
func IdentitySubnet(ip net.IP) AllowedIP {
	return AllowedIP{
		IP:   ip.To16(),
		Mask: net.CIDRMask(128, 128),
	}
}

// Errno is an error returned by the IPC.
type Errno = value.Int64[key.Errno]

const (
	// ErrnoNone means no error occurred
	ErrnoNone = Errno(0)
	// ErrnoIO references [ipc.IpcErrorIO].
	ErrnoIO = Errno(ipc.IpcErrorIO)
	// ErrnoProtocol references [ipc.IpcErrorProtocol].
	ErrnoProtocol = Errno(ipc.IpcErrorProtocol)
	// ErrnoInvalid references [ipc.IpcErrorInvalid].
	ErrnoInvalid = Errno(ipc.IpcErrorInvalid)
	// ErrnoPortInUse references [ipc.IpcErrorPortInUse].
	ErrnoPortInUse = Errno(ipc.IpcErrorPortInUse)
	// ErrnoUnknown references [ipc.IpcErrorUnknown].
	ErrnoUnknown = Errno(int64(ipc.IpcErrorUnknown))
)
