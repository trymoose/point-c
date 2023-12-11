package configvalues

import "net"

// These are convenience types that may be used in configurations.
type (
	// Port defines a network port value, which is an unsigned 16-bit integer.
	// The validity of 0 as a port number depends on the specific use case or context.
	Port = CaddyTextUnmarshaler[uint16, ValueUnsigned[uint16], *ValueUnsigned[uint16]]

	// UDPAddr is a type alias for handling UDP network addresses.
	// It wraps the [net.UDPAddr] type and utilizes [CaddyTextUnmarshaler] for parsing
	// and handling UDP addresses in text form.
	UDPAddr = CaddyTextUnmarshaler[*net.UDPAddr, ValueUDPAddr, *ValueUDPAddr]

	// IP is a type alias for handling IP addresses.
	// It wraps the [net.IP] type and uses [CaddyTextUnmarshaler] for converting text-based
	// IPv4 or IPv6 address representations into [net.IP].
	IP = CaddyTextUnmarshaler[net.IP, ValueIP, *ValueIP]

	// Hostname represents a unique hostname string.
	// This type uses [CaddyTextUnmarshaler] with a string base type.
	Hostname = CaddyTextUnmarshaler[string, ValueString, *ValueString]
)
