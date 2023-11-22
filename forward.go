package point_c

import "github.com/caddyserver/caddy/v2"

var (
	_ caddy.Provisioner  = (*Forward)(nil)
	_ caddy.CleanerUpper = (*Forward)(nil)
	_ caddy.Module       = (*Forward)(nil)
	_ caddy.App          = (*Forward)(nil)
)

type Forward struct {
}

func (p *Forward) Start() error {
	//TODO implement me
	panic("implement me")
}

func (p *Forward) Stop() error {
	//TODO implement me
	panic("implement me")
}

func (p *Forward) Cleanup() error {
	//TODO implement me
	panic("implement me")
}

func (*Forward) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "forward",
		New: func() caddy.Module { return new(Forward) },
	}
}

func (p *Forward) Provision(context caddy.Context) error {
	//TODO implement me
	panic("implement me")
}
