package module

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	pointc "github.com/trymoose/point-c"
	"github.com/trymoose/point-c/module/internal"
)

const CaddyfileForwardName = "forward"

func init() {
	caddy.RegisterModule(new(pointc.Forwards))
	httpcaddyfile.RegisterGlobalOption(CaddyfileForwardName, internal.Unmarshaler[pointc.Forwards, *pointc.Forwards](CaddyfileForwardName))
}
