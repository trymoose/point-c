package point_c

import (
	"context"
	"github.com/trymoose/point-c/pkg/channel-listener"
	"net"
)

// StubListener is a listener that blocks on [net.Listener.Accept] until [net.Listener.Close] is called.
func StubListener(_ context.Context, _, addr string, _ net.ListenConfig) (any, error) {
	return channel_listener.New(make(<-chan net.Conn), stubAddr(addr)), nil
}

type stubAddr string

func (stubAddr) Network() string  { return "stub" }
func (d stubAddr) String() string { return string(d) }
