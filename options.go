package pointc

import (
	"errors"
	"github.com/trymoose/point-c/wgapi"
	"github.com/trymoose/point-c/wglog"
)

type (
	option  func(*options) error
	options struct {
		tun     Device
		bind    Bind
		loggers []*wglog.Logger
		cfg     *wgapi.Configurable
		closer  []func() error
	}
)

func (o *options) apply(opts []option) error {
	for _, opt := range opts {
		if err := opt(o); err != nil {
			return err
		}
	}
	return nil
}

func OptionNop() option { return func(*options) error { return nil } }

func OptionErr(e error) option { return func(*options) error { return e } }

func OptionDevice(d Device) option {
	if d == nil {
		return OptionNop()
	}
	return func(o *options) error { o.tun = d; return nil }
}

func OptionBind(b Bind) option {
	if b == nil {
		return OptionNop()
	}
	return func(o *options) error { o.bind = b; return nil }
}

func OptionLogger(l *wglog.Logger) option {
	if l == nil {
		return OptionNop()
	}
	return func(o *options) error { o.loggers = append(o.loggers, l); return nil }
}

func OptionConfig(cfg wgapi.Configurable) option {
	if cfg == nil {
		return OptionNop()
	}
	return func(o *options) error { o.cfg = &cfg; return nil }
}

// OptionNetDevice initializes a userspace networking stack.
// Functions for using the network will be stored at the location given.
// Since this may produce errors, *p is only valid if [New] returns a nil error.
func OptionNetDevice(p **Net) option {
	if p == nil {
		return OptionErr(errors.New("invalid net pointer"))
	}
	return func(o *options) error {
		n, err := NewDefaultNetstack()
		if err != nil {
			return err
		}
		o.tun = n
		o.closer = append(o.closer, n.Close)
		nn := n.Net()
		*p = nn
		return nil
	}
}
