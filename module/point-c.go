package module

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	pointc "github.com/trymoose/point-c"
	"github.com/trymoose/point-c/module/internal"
)

func init() {
	caddy.RegisterModule(new(pointc.Pointc))
	httpcaddyfile.RegisterGlobalOption("point-c", internal.Unmarshaler[pointc.Pointc, *pointc.Pointc]("point-c"))
}
