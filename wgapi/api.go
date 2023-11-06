package wgapi

import (
	"bytes"
	"fmt"
	"github.com/trymoose/wg4d/wgapi/internal/core"
	"github.com/trymoose/wg4d/wgapi/internal/key"
	"github.com/trymoose/wg4d/wgapi/internal/value"
	"github.com/trymoose/wg4d/wgapi/internal/value/wgkey"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"io"
	"net"
)

type (
	Configurable interface {
		WGConfig() io.Reader
	}

	IPCKeyValue = core.KeyValue
)

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
	KeyPublic    = wgkey.Key[wgkey.Public]
	KeyPrivate   = wgkey.Key[wgkey.Private]
	KeyPreShared = wgkey.Key[wgkey.PreShared]
)

func NewKey() (*KeyPrivate, error) {
	k, err := wgtypes.GeneratePrivateKey()
	return (*KeyPrivate)(&k), err
}

func NewPreShared() (*KeyPreShared, error) {
	k, err := wgtypes.GenerateKey()
	return (*KeyPreShared)(&k), err
}

type (
	PrivateKey   = value.Key[key.PrivateKey, wgkey.Private]
	PublicKey    = value.Key[key.PublicKey, wgkey.Public]
	PresharedKey = value.Key[key.PresharedKey, wgkey.PreShared]
)

const DefaultPersistentKeepalive PersistentKeepalive = 25

type PersistentKeepalive = value.Uint16[key.PersistentKeepalive]

type (
	ReplacePeers      = value.True[key.ReplacePeers]
	Remove            = value.True[key.Remove]
	UpdateOnly        = value.True[key.UpdateOnly]
	ReplaceAllowedIPs = value.True[key.ReplaceAllowedIPs]
	ProtocolVersion   = value.One[key.ProtocolVersion]
	Get               = value.One[key.Get]
	Set               = value.One[key.Set]
)

type (
	LastHandshakeTimeSec  = value.Int64[key.LastHandshakeTimeSec]
	LastHandshakeTimeNSec = value.Int64[key.LastHandshakeTimeNSec]
)

type (
	RXBytes = value.Uint64[key.RXBytes]
	TXBytes = value.Uint64[key.TXBytes]
)

type (
	Endpoint   = value.UDPAddr[key.Endpoint]
	AllowedIP  = value.IPNet[key.AllowedIP]
	ListenPort = value.Uint16[key.ListenPort]
	FWMark     = value.Uint32[key.FWMark]
)

var EmptySubnet = func() AllowedIP {
	const allSubnets = "0.0.0.0/0"
	_, ip, err := net.ParseCIDR(allSubnets)
	if err != nil {
		panic(fmt.Errorf("failed to parse %q into %T: %w", allSubnets, ip, err))
	}
	return AllowedIP(*ip)
}()

func IdentitySubnet(ip net.IP) AllowedIP {
	return AllowedIP{
		IP:   ip,
		Mask: net.CIDRMask(128, 128),
	}
}

type Errno = value.Int64[key.Errno]

const (
	ErrnoNone      = Errno(0)
	ErrnoIO        = Errno(ipc.IpcErrorIO)
	ErrnoProtocol  = Errno(ipc.IpcErrorProtocol)
	ErrnoInvalid   = Errno(ipc.IpcErrorInvalid)
	ErrnoPortInUse = Errno(ipc.IpcErrorPortInUse)
	ErrnoUnknown   = Errno(int64(ipc.IpcErrorUnknown))
)
