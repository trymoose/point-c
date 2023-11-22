package configvalues

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/caddyserver/caddy/v2"
	"github.com/tidwall/gjson"
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
	valueConstraint[V, T any] interface {
		*T
		Value[V]
	}
	CaddyTextUnmarshaler[V, T any, TP valueConstraint[V, T]] struct {
		value    T
		original string
	}
)

type (
	// Port is a value in the uint16 range. 0 may or may nor be valid depending on the context.
	Port = CaddyTextUnmarshaler[uint16, valueUnsigned[uint16], *valueUnsigned[uint16]]
	//PortPair = configvalues.CaddyTextUnmarshaler[[2]uint16, configvalues.PortPair, *configvalues.PortPair]

	// UDPAddr is a wrapper for the [net.UDPAddr] type.
	UDPAddr = CaddyTextUnmarshaler[*net.UDPAddr, valueUDPAddr, *valueUDPAddr]
	// IP is wrapper for the [net.IP] type.
	IP = CaddyTextUnmarshaler[net.IP, valueIP, *valueIP]

	// Hostname is a unique hostname.
	Hostname = CaddyTextUnmarshaler[string, valueString, *valueString]
)

func NewCaddyTextUnmarshaler[V, T any, TP valueConstraint[V, T]](text string) (*CaddyTextUnmarshaler[V, T, TP], error) {
	var c CaddyTextUnmarshaler[V, T, TP]
	if err := c.UnmarshalText([]byte(text)); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c CaddyTextUnmarshaler[V, T, TP]) MarshalText() (text []byte, err error) {
	return []byte(c.original), nil
}

var caddyReplacer = sync.OnceValue(caddy.NewReplacer)

func (c *CaddyTextUnmarshaler[V, T, TP]) UnmarshalText(text []byte) error {
	c.original = string(text)
	text = []byte(caddyReplacer().ReplaceAll(c.original, ""))
	return any(&c.value).(encoding.TextUnmarshaler).UnmarshalText(text)
}

func (c *CaddyTextUnmarshaler[V, T, TP]) MarshalJSON() (text []byte, err error) {
	text, err = c.MarshalText()
	if !gjson.ValidBytes(text) {
		text = []byte(strconv.Quote(string(text)))
	}
	return
}

func (c *CaddyTextUnmarshaler[V, T, TP]) UnmarshalJSON(text []byte) error {
	if s := ""; json.Unmarshal(text, &s) == nil {
		text = []byte(s)
	}
	return c.UnmarshalText(text)
}

func (c *CaddyTextUnmarshaler[V, T, TP]) Value() V {
	return any(&c.value).(Value[V]).Value()
}

type valueString string

func (s *valueString) UnmarshalText(b []byte) error {
	*s = valueString(b)
	return nil
}

func (s *valueString) Value() string {
	return string(*s)
}

type valueUnsigned[N constraints.Unsigned] struct{ V N }

func (n *valueUnsigned[N]) UnmarshalText(b []byte) error {
	i, err := strconv.ParseUint(string(b), 10, binary.Size(n.V)*8)
	if err != nil {
		return err
	}
	n.V = N(i)
	return nil
}

func (n *valueUnsigned[N]) Value() N {
	return n.V
}

type valueUDPAddr net.UDPAddr

func (addr *valueUDPAddr) UnmarshalText(text []byte) error {
	a, err := net.ResolveUDPAddr("udp", string(text))
	if err != nil {
		return err
	}
	*addr = (valueUDPAddr)(*a)
	return nil
}

func (addr *valueUDPAddr) Value() *net.UDPAddr {
	return (*net.UDPAddr)(addr)
}

type valueIP net.IP

func (ip *valueIP) UnmarshalText(text []byte) error {
	return ((*net.IP)(ip)).UnmarshalText(text)
}

func (ip *valueIP) Value() net.IP {
	return net.IP(*ip)
}

type valuePortPair [2]valueUnsigned[uint16]

func (pp *valuePortPair) UnmarshalText(b []byte) error {
	src, dst, ok := bytes.Cut(b, []byte{':'})
	if !ok {
		return errors.New("not a port:port pair")
	}
	if err := errors.Join(pp[0].UnmarshalText(src), pp[1].UnmarshalText(dst)); err != nil {
		return err
	}
	return nil
}

func (pp valuePortPair) Value() [2]uint16 { return [2]uint16{pp[0].Value(), pp[1].Value()} }
