package module

import (
	"github.com/caddyserver/caddy/v2"
	pointc "github.com/trymoose/point-c"
)

func init() {
	caddy.RegisterNetwork("stub", pointc.StubListener)
}
