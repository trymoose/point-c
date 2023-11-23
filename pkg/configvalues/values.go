package configvalues

import (
	"encoding/binary"
	"golang.org/x/exp/constraints"
	"net"
	"strconv"
)

type ValueBool bool

func (b *ValueBool) UnmarshalText(text []byte) error {
	bb, err := strconv.ParseBool(string(text))
	if err != nil {
		return err
	}
	*b = ValueBool(bb)
	return nil
}

func (b *ValueBool) Value() bool {
	return bool(*b)
}

type ValueString string

func (s *ValueString) UnmarshalText(b []byte) error {
	*s = ValueString(b)
	return nil
}

func (s *ValueString) Value() string {
	return string(*s)
}

type ValueUnsigned[N constraints.Unsigned] struct{ V N }

func (n *ValueUnsigned[N]) UnmarshalText(b []byte) error {
	i, err := strconv.ParseUint(string(b), 10, binary.Size(n.V)*8)
	if err != nil {
		return err
	}
	n.V = N(i)
	return nil
}

func (n *ValueUnsigned[N]) Value() N {
	return n.V
}

type ValueUDPAddr net.UDPAddr

func (addr *ValueUDPAddr) UnmarshalText(text []byte) error {
	a, err := net.ResolveUDPAddr("udp", string(text))
	if err != nil {
		return err
	}
	*addr = (ValueUDPAddr)(*a)
	return nil
}

func (addr *ValueUDPAddr) Value() *net.UDPAddr {
	return (*net.UDPAddr)(addr)
}

type ValueIP net.IP

func (ip *ValueIP) UnmarshalText(text []byte) error {
	return ((*net.IP)(ip)).UnmarshalText(text)
}

func (ip *ValueIP) Value() net.IP {
	return net.IP(*ip)
}
