package module

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	pointc "github.com/trymoose/point-c"
	"github.com/trymoose/point-c/module/internal"
)

const CaddyfilePointCName = "point-c"

func init() {
	caddy.RegisterModule(new(pointc.Pointc))
	httpcaddyfile.RegisterGlobalOption(CaddyfilePointCName, internal.Unmarshaler[pointc.Pointc, *pointc.Pointc](CaddyfilePointCName))
}
