package point_c

import (
	"context"
	"github.com/caddyserver/caddy/v2"
	"github.com/trymoose/point-c/pkg/channel-listener"
	"net"
)

func init() {
	caddy.RegisterNetwork("stub", func(_ context.Context, _, addr string, _ net.ListenConfig) (any, error) {
		return channel_listener.New(make(<-chan net.Conn), stubAddr(addr)), nil
	})
}

type stubAddr string

func (stubAddr) Network() string  { return "stub" }
func (d stubAddr) String() string { return string(d) }
