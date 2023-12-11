package configvalues

import (
	"encoding/binary"
	"golang.org/x/exp/constraints"
	"net"
	"strconv"
	"unsafe"
)

// ValueBool handles unmarshalling bool values.
type ValueBool bool

// UnmarshalText parses the bool with [strconv.ParseBool] internally.
func (b *ValueBool) UnmarshalText(text []byte) error {
	bb, err := strconv.ParseBool(string(text))
	if err != nil {
		return err
	}
	*b = ValueBool(bb)
	return nil
}

// Value returns the underlying bool value of ValueBool.
func (b *ValueBool) Value() bool {
	return bool(*b)
}

// ValueString handles unmarshalling string values.
type ValueString string

// UnmarshalText just sets the value to string(b).
func (s *ValueString) UnmarshalText(b []byte) error {
	*s = ValueString(b)
	return nil
}

// Value returns the underlying string value of ValueString.
func (s *ValueString) Value() string {
	return string(*s)
}

// ValueUnsigned is a generic type for unmarshalling an unsigned number.
// N must be an unsigned type (e.g., uint, uint32).
type ValueUnsigned[N constraints.Unsigned] struct{ V N }

// UnmarshalText parses the uint with [strconv.ParseUint] internally.
func (n *ValueUnsigned[N]) UnmarshalText(b []byte) error {
	var size int
	switch any(n.V).(type) {
	// uintptr and uint report -8 with binary.Size
	case uintptr, uint:
		size = int(unsafe.Sizeof(n.V))
	default:
		size = binary.Size(n.V)
	}

	i, err := strconv.ParseUint(string(b), 10, size*8)
	if err != nil {
		return err
	}
	n.V = N(i)
	return nil
}

// Value returns the underlying unsigned number of ValueUnsigned.
func (n *ValueUnsigned[N]) Value() N {
	return n.V
}

// ValueUDPAddr handles unmarshalling a [net.UDPAddr].
type ValueUDPAddr net.UDPAddr

// UnmarshalText implements the unmarshaling of text data into a UDP address.
// It resolves the text using [net.ResolveUDPAddr].
func (addr *ValueUDPAddr) UnmarshalText(text []byte) error {
	a, err := net.ResolveUDPAddr("udp", string(text))
	if err != nil {
		return err
	}
	*addr = (ValueUDPAddr)(*a)
	return nil
}

// Value returns the underlying net.UDPAddr of ValueUDPAddr.
func (addr *ValueUDPAddr) Value() *net.UDPAddr {
	return (*net.UDPAddr)(addr)
}

// ValueIP handles unmarshalling [net.IP].
type ValueIP net.IP

// UnmarshalText implements the unmarshaling of text data into an IP address.
// It delegates to the [encoding.TextUnmarshaler] implementation of [net.IP].
func (ip *ValueIP) UnmarshalText(text []byte) error {
	return ((*net.IP)(ip)).UnmarshalText(text)
}

// Value returns the underlying net.IP of ValueIP.
func (ip *ValueIP) Value() net.IP {
	return net.IP(*ip)
}
