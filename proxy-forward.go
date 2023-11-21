package point_c

var (
	_ caddy.Provisioner  = (*ProxyForward)(nil)
	_ caddy.CleanerUpper = (*ProxyForward)(nil)
	_ caddy.Module       = (*ProxyForward)(nil)
	_ caddy.App          = (*ProxyForward)(nil)
)

type ProxyForward struct {
}

func (p *ProxyForward) Start() error {
	//TODO implement me
	panic("implement me")
}

func (p *ProxyForward) Stop() error {
	//TODO implement me
	panic("implement me")
}

func (p *ProxyForward) Cleanup() error {
	//TODO implement me
	panic("implement me")
}

func (*ProxyForward) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "proxy-forward",
		New: func() caddy.Module { return new(ProxyForward) },
	}
}

func (p *ProxyForward) Provision(context caddy.Context) error {
	//TODO implement me
	panic("implement me")
}
