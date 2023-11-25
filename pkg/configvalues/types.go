package configvalues

import "net"

type (
	// Port is a value in the uint16 range. 0 may or may nor be valid depending on the context.
	Port = CaddyTextUnmarshaler[uint16, ValueUnsigned[uint16], *ValueUnsigned[uint16]]

	// UDPAddr is a wrapper for the [net.UDPAddr] type.
	UDPAddr = CaddyTextUnmarshaler[*net.UDPAddr, ValueUDPAddr, *ValueUDPAddr]
	// IP is wrapper for the [net.IP] type.
	IP = CaddyTextUnmarshaler[net.IP, ValueIP, *ValueIP]

	// Hostname is a unique hostname.
	Hostname = CaddyTextUnmarshaler[string, ValueString, *ValueString]
)
