package wg4d

import (
	"context"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/arp"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"net"
	"os"
	"slices"
	"sync"
)

const (
	DefaultMTU         = device.DefaultMTU
	DefaultBatchSize   = 1
	DefaultChannelSize = 1024
)

type Netstack struct {
	ep         *channel.Endpoint
	stack      *stack.Stack
	events     chan tun.Event
	batchSize  int
	close      sync.Once
	closeErr   error
	done       chan struct{}
	read       chan []byte
	defaultNIC tcpip.NICID
	mtu        int
}

func NewNetstack(mtu int, batchSize int, channelSize int) (*Netstack, error) {
	ns := &Netstack{
		mtu: mtu,
		ep:  channel.New(channelSize, uint32(mtu), ""),
		stack: stack.New(stack.Options{
			NetworkProtocols: []stack.NetworkProtocolFactory{
				ipv4.NewProtocol,
				ipv6.NewProtocol,
				arp.NewProtocol},
			TransportProtocols: []stack.TransportProtocolFactory{
				tcp.NewProtocol,
				udp.NewProtocol,
				icmp.NewProtocol4,
				icmp.NewProtocol6},
			HandleLocal: false,
		}),
		events:    make(chan tun.Event, 1),
		batchSize: batchSize,
		done:      make(chan struct{}),
		read:      make(chan []byte),
	}
	ns.ep.AddNotify((*writeNotify)(ns))

	var enableSACK tcpip.TCPSACKEnabled = true
	if err := ns.stack.SetTransportProtocolOption(tcp.ProtocolNumber, &enableSACK); err != nil {
		return nil, &TCPIPError{Err: err}
	}

	ns.defaultNIC = tcpip.NICID(ns.stack.UniqueID())
	if err := ns.stack.CreateNICWithOptions(ns.defaultNIC, ns.ep, stack.NICOptions{Name: ""}); err != nil {
		return nil, &TCPIPError{Err: err}
	}

	ns.stack.SetSpoofing(ns.defaultNIC, true)
	ns.stack.SetPromiscuousMode(ns.defaultNIC, true)
	ns.stack.AddRoute(tcpip.Route{Destination: header.IPv4EmptySubnet, NIC: ns.defaultNIC})
	ns.stack.AddRoute(tcpip.Route{Destination: header.IPv6EmptySubnet, NIC: ns.defaultNIC})

	ns.events <- tun.EventUp
	return ns, nil
}

func (ns *Netstack) Device() tun.Device { return (*dev)(ns) }

func (ns *Netstack) Close() error {
	ns.close.Do(func() {
		close(ns.done)
		go func() { ns.events <- tun.EventDown }()
		ns.ep.Close()
		ns.ep.Wait()
	})
	return ns.closeErr
}

var _ channel.Notification = (*writeNotify)(nil)

type writeNotify Netstack

func (w *writeNotify) WriteNotify() {
	pkt := w.ep.Read()
	if pkt.IsNil() {
		return
	}

	view := slices.Clone(pkt.ToView().AsSlice())
	pkt.DecRef()
	select {
	case <-w.done:
	case w.read <- view:
	}
}

var _ tun.Device = (*dev)(nil)

type dev Netstack

func (d *dev) File() *os.File           { return nil }
func (d *dev) Name() (string, error)    { return "netstack", nil }
func (d *dev) MTU() (int, error)        { return d.mtu, nil }
func (d *dev) Events() <-chan tun.Event { return d.events }
func (d *dev) BatchSize() int           { return d.batchSize }
func (d *dev) Close() error             { return ((*Netstack)(d)).Close() }

func (d *dev) Read(buf [][]byte, sizes []int, offset int) (n int, err error) {
	select {
	case <-d.done:
		return 0, os.ErrClosed
	case p := <-d.read:
		sizes[0] = copy(buf[0][offset:], p)
		return 1, nil
	}
}

func (d *dev) Write(buf [][]byte, offset int) (int, error) {
	for _, buf := range buf {
		buf = buf[offset:]
		if len(buf) == 0 {
			continue
		}

		packet := stack.NewPacketBuffer(stack.PacketBufferOptions{Payload: buffer.MakeWithData(buf)})
		switch buf[0] >> 4 {
		case 4:
			d.ep.InjectInbound(header.IPv4ProtocolNumber, packet)
		case 6:
			d.ep.InjectInbound(header.IPv6ProtocolNumber, packet)
		}
	}
	return len(buf), nil
}

type TCPIPError struct{ Err tcpip.Error }

func (err *TCPIPError) Error() string { return err.Err.String() }

type Net struct {
	stack *stack.Stack
	local tcpip.FullAddress
	nic   tcpip.NICID
}

func (ns *Netstack) Net(local net.IP) *Net {
	return &Net{
		stack: ns.stack,
		local: tcpip.FullAddress{
			NIC:  ns.defaultNIC,
			Addr: tcpip.AddrFromSlice(local.To4()),
		},
		nic: ns.defaultNIC,
	}
}

func (n *Net) ListenTCP(port uint16) (net.Listener, error) {
	return gonet.ListenTCP(n.stack, tcpip.FullAddress{
		NIC:  n.local.NIC,
		Addr: n.local.Addr,
		Port: port,
	}, ipv4.ProtocolNumber)
}

func (n *Net) DialTCP(ctx context.Context, addr *net.TCPAddr) (net.Conn, error) {
	return gonet.DialTCPWithBind(ctx, n.stack, n.local, tcpip.FullAddress{
		NIC:  n.nic,
		Addr: tcpip.AddrFromSlice(addr.IP.To4()),
		Port: uint16(addr.Port),
	}, ipv4.ProtocolNumber)
}

func (n *Net) DialUDP(addr *net.UDPAddr) (net.PacketConn, error) {
	return gonet.DialUDP(n.stack, &n.local, &tcpip.FullAddress{
		NIC:  n.nic,
		Addr: tcpip.AddrFromSlice(addr.IP.To4()),
		Port: uint16(addr.Port),
	}, ipv4.ProtocolNumber)
}
