package point_c

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/trymoose/point-c/pkg/configvalues"
	"github.com/trymoose/point-c/pkg/resuming"
	"go.mrchanchal.com/zaphandler"
	"io"
	"log/slog"
	"net"
	"slices"
	"strconv"
	"strings"
	"sync"
)

func init() {
	caddy.RegisterModule(new(Forwards))
	httpcaddyfile.RegisterGlobalOption("forward", resuming.Unmarshaler[Forwards, *Forwards]("forward"))
}

var (
	_ caddy.Provisioner     = (*Forwards)(nil)
	_ caddy.CleanerUpper    = (*Forwards)(nil)
	_ caddy.Module          = (*Forwards)(nil)
	_ caddy.App             = (*Forwards)(nil)
	_ caddyfile.Unmarshaler = (*Forwards)(nil)
)

type (
	Forwards struct {
		Forwards   []*Forward `json:"forwards,omitempty"`
		forwarders []*Forwarder
		stop       []func() error
		logger     *slog.Logger
	}
	Forward struct {
		Name  configvalues.Hostname
		Ports []*configvalues.CaddyTextUnmarshaler[*PortPair, PortPair, *PortPair]
	}
)

// PortPair is a [<host>:]<src>:<dst>[/<tcp|udp>] port pair.
type PortPair struct {
	src, dst configvalues.Port
	proto    configvalues.CaddyTextUnmarshaler[string, configvalues.ValueString, *configvalues.ValueString]
	host     *configvalues.IP
}

func (pp *PortPair) Src() uint16 { return pp.src.Value() }
func (pp *PortPair) Dst() uint16 { return pp.dst.Value() }
func (pp *PortPair) IsUDP() bool { return pp.proto.Value() == "udp" }
func (pp *PortPair) Host() (net.IP, bool) {
	if pp.host == nil {
		return nil, false
	}
	return pp.host.Value(), true
}

func (pp *PortPair) UnmarshalText(b []byte) error {
	src, dst, ok := bytes.Cut(b, []byte{':'})
	if !ok {
		return errors.New("not a port:port pair")
	}

	host := src
	src, dst, ok = bytes.Cut(dst, []byte{':'})
	if ok {
		pp.host = new(configvalues.IP)
		if err := pp.host.UnmarshalText(host); err != nil {
			return err
		}
	} else {
		pp.host = nil
		src, dst = host, src
	}

	dst, proto, ok := bytes.Cut(dst, []byte{'/'})
	if ok {
		if err := pp.proto.UnmarshalText(proto); err != nil {
			return err
		}
		if proto := pp.proto.Value(); !slices.Contains([]string{"tcp", "udp"}, proto) {
			return fmt.Errorf("unrecognized protocol %q", proto)
		}
	}

	if err := errors.Join(pp.src.UnmarshalText(src), pp.dst.UnmarshalText(dst)); err != nil {
		return err
	}
	return nil
}

func (pp *PortPair) Value() *PortPair { return pp }

func (p *Forwards) Start() error {
	for _, f := range p.forwarders {
		anyLn, err := f.Addr.Listen(f.Ctx, 0, net.ListenConfig{})
		if err != nil {
			return err
		}
		ln := anyLn.(net.Listener)
		p.stop = append(p.stop, ln.Close)

		ctx, cancel := context.WithCancel(f.Ctx)
		p.stop = append(p.stop, func() error { cancel(); return nil })

		go f.forwardListener(ctx, cancel, ln, p.logger)
	}
	return nil
}

func (f *Forwarder) forwardListener(ctx context.Context, cancel context.CancelFunc, ln net.Listener, logger *slog.Logger) {
	defer cancel()
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		ctx, cancel := context.WithCancel(ctx)
		context.AfterFunc(ctx, func() { c.Close() })
		go func() {
			defer cancel()
			remote := c.RemoteAddr().(*net.TCPAddr).IP
			d := f.Net.Dialer(remote, 0)

			rc, err := d.Dial(ctx, &net.TCPAddr{IP: f.Net.LocalAddr(), Port: int(f.Dst)})
			if err != nil {
				logger.Error("failed to dial remote in tunnel", "local", remote, "remote", f.Net.LocalAddr(), "port", f.Dst)
				return
			}
			context.AfterFunc(ctx, func() { rc.Close() })

			var wg sync.WaitGroup
			done := func() func() { wg.Add(1); return func() { defer wg.Done(); defer cancel() } }
			go tcpCopy(done(), rc, c, logger)
			go tcpCopy(done(), c, rc, logger)
			wg.Wait()
		}()
	}
}

func tcpCopy(done func(), dst io.Writer, src io.Reader, logger *slog.Logger) {
	defer done()
	if _, err := io.Copy(dst, src); err != nil {
		logger.Error("error copying data between connections", "error", err)
	}
}

func (p *Forwards) Stop() error {
	return p.stopAll()
}

func (p *Forwards) Cleanup() error {
	p.forwarders = nil
	return p.stopAll()
}

func (p *Forwards) stopAll() (err error) {
	var fn func() error
	slices.Reverse(p.stop)
	for len(p.stop) > 0 {
		fn, p.stop = p.stop[0], p.stop[1:]
		err = errors.Join(err, fn())
	}
	return
}

func (*Forwards) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "forward",
		New: func() caddy.Module { return new(Forwards) },
	}
}

type Forwarder struct {
	Net  Net
	Dst  uint16
	Addr caddy.NetworkAddress
	Ctx  caddy.Context
}

func (p *Forwards) Provision(ctx caddy.Context) error {
	p.logger = slog.New(zaphandler.New(ctx.Logger()))
	v, err := ctx.App("point-c")
	if err != nil {
		return err
	}
	pc := v.(*Pointc)

	var addrStr strings.Builder
	for _, fwd := range p.Forwards {
		n, ok := pc.Lookup(fwd.Name.Value())
		if !ok {
			return fmt.Errorf("network %q not found", fwd.Name.Value())
		}

		for _, pp := range fwd.Ports {
			if pp.Value().IsUDP() {
				udpForwarder(p.logger)
			} else {
				f, err := tcpForwarder(ctx, n, pp.Value(), &addrStr)
				if err != nil {
					return err
				}
				p.forwarders = append(p.forwarders, f)
			}
		}
	}

	p.Forwards = nil
	return nil
}

var warnNoUDPForwarderOnce sync.Once

func udpForwarder(logger *slog.Logger) {
	warnNoUDPForwarderOnce.Do(func() { logger.Warn("udp forwarding not supported") })
}

func tcpForwarder(ctx caddy.Context, n Net, pp *PortPair, addrStr *strings.Builder) (_ *Forwarder, err error) {
	f := Forwarder{
		Net: n,
		Dst: pp.Dst(),
		Ctx: ctx,
	}

	addrStr.Reset()
	if pp.IsUDP() {
		addrStr.WriteString("udp/")
	}
	if h, ok := pp.Host(); ok {
		addrStr.WriteString(h.String())
	}
	addrStr.WriteRune(':')
	addrStr.Write(strconv.AppendInt(nil, int64(pp.Src()), 10))

	f.Addr, err = caddy.ParseNetworkAddress(addrStr.String())
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// UnmarshalCaddyfile unmarshals a submodules from a caddyfile.
//
//	{
//	  forward <net name> {
//	    [<host>:]<src>:<dst>[/<tcp|udp>]
//	  }
//	}
func (p *Forwards) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		var f Forward
		if !d.NextArg() {
			return d.ArgErr()
		} else if err := f.Name.UnmarshalText([]byte(d.Val())); err != nil {
			return err
		}

		for nesting := d.Nesting(); d.NextBlock(nesting); {
			if !d.NextArg() {
				continue
			}
			var pp configvalues.CaddyTextUnmarshaler[*PortPair, PortPair, *PortPair]
			if err := pp.UnmarshalText([]byte(d.Val())); err != nil {
				return err
			}
			f.Ports = append(f.Ports, &pp)
		}

		p.Forwards = append(p.Forwards, &f)
	}
	return nil
}
