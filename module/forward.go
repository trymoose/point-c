package module

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	pointc "github.com/trymoose/point-c"
	"github.com/trymoose/point-c/module/internal"
)

func init() {
	caddy.RegisterModule(new(pointc.Forwards))
	httpcaddyfile.RegisterGlobalOption("forward", internal.Unmarshaler[pointc.Forwards, *pointc.Forwards]("forward"))
}
