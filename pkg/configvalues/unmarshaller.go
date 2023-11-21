package configvalues

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"
	"github.com/caddyserver/caddy/v2"
	"golang.org/x/exp/constraints"
	"net"
	"strconv"
	"sync"
)

type (
	Value[V any] interface {
		encoding.TextUnmarshaler
		Value() V
	}
	valuePtr[V, T any] interface {
		*T
		Value[V]
	}
	CaddyTextUnmarshaler[V, T any, TP valuePtr[V, T]] struct {
		value    T
		original string
	}
)

func NewCaddyTextUnmarshaler[V, T any, TP valuePtr[V, T]](text string) (*CaddyTextUnmarshaler[V, T, TP], error) {
	var c CaddyTextUnmarshaler[V, T, TP]
	if err := c.UnmarshalText([]byte(text)); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c CaddyTextUnmarshaler[V, T, TP]) MarshalText() (text []byte, err error) {
	return []byte(c.original), nil
}

var replacer = sync.OnceValue(caddy.NewReplacer)

func (c *CaddyTextUnmarshaler[V, T, TP]) UnmarshalText(text []byte) error {
	c.original = string(text)
	text = []byte(replacer().ReplaceAll(c.original, ""))
	return any(&c.value).(encoding.TextUnmarshaler).UnmarshalText(text)
}

func (c *CaddyTextUnmarshaler[V, T, TP]) Value() V {
	return any(&c.value).(Value[V]).Value()
}

type String string

func (s *String) UnmarshalText(b []byte) error {
	*s = String(b)
	return nil
}

func (s *String) Value() string {
	return string(*s)
}

type Unsigned[N constraints.Unsigned] struct{ V N }

func (n *Unsigned[N]) UnmarshalText(b []byte) error {
	i, err := strconv.ParseUint(string(b), 10, binary.Size(n.V)*8)
	if err != nil {
		return err
	}
	n.V = N(i)
	return nil
}

func (n *Unsigned[N]) Value() N {
	return n.V
}

type UDPAddr net.UDPAddr

func (addr *UDPAddr) UnmarshalText(text []byte) error {
	a, err := net.ResolveUDPAddr("udp", string(text))
	if err != nil {
		return err
	}
	*addr = (UDPAddr)(*a)
	return nil
}

func (addr *UDPAddr) Value() *net.UDPAddr {
	return (*net.UDPAddr)(addr)
}

type IP net.IP

func (ip *IP) UnmarshalText(text []byte) error {
	return ((*net.IP)(ip)).UnmarshalText(text)
}

func (ip *IP) Value() net.IP {
	return net.IP(*ip)
}

type PortPair [2]Unsigned[uint16]

func (pp *PortPair) UnmarshalText(b []byte) error {
	src, dst, ok := bytes.Cut(b, []byte{':'})
	if !ok {
		return errors.New("not a port:port pair")
	}
	if err := errors.Join(pp[0].UnmarshalText(src), pp[1].UnmarshalText(dst)); err != nil {
		return err
	}
	return nil
}

func (pp PortPair) Value() [2]uint16 { return [2]uint16{pp[0].Value(), pp[1].Value()} }
