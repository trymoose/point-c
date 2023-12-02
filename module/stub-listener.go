package module

import (
	"github.com/caddyserver/caddy/v2"
	pointc "github.com/trymoose/point-c"
)

const CaddyNetworkStubName = "stub"

func init() {
	caddy.RegisterNetwork(CaddyNetworkStubName, pointc.StubListener)
}
