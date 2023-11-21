package wg

import (
	"errors"
	"github.com/trymoose/point-c/wg/api"
	"github.com/trymoose/point-c/wg/log"
)

type (
	option func(*options) error
	// options holds settings for an interface.
	options struct {
		tun     Device              // tun is a required network for unencrypted traffic.
		bind    Bind                // bind is the network for encrypted traffic.
		loggers []*wglog.Logger     // loggers is a slice of Logger instances for logging purposes.
		cfg     *wgapi.Configurable // cfg is an initial IPC configuration.
		closer  []func() error      // closer is the resources that need to be cleaned up.
	}
)

// apply applies a slice of [option] functions to the [options] struct.
func (o *options) apply(opts []option) error {
	for _, opt := range opts {
		// If an option function returns an error, it is immediately returned to the caller.
		if err := opt(o); err != nil {
			return err
		}
	}
	return nil
}

// cleanUp closes any open resources in the event of a failure.
func (o *options) cleanUp(failed *bool, err *error) {
	if *failed {
		for _, c := range o.closer {
			*err = errors.Join(*err, c())
		}
	}
}

// OptionNop is an option function that does nothing. Useful as a placeholder.
func OptionNop() option { return func(*options) error { return nil } }

// OptionErr causes [New] to fail with the given error
func OptionErr(e error) option { return func(*options) error { return e } }

// OptionDevice specifies the Device in the options struct.
func OptionDevice(d Device) option {
	if d == nil {
		return OptionNop()
	}
	return func(o *options) error { o.tun = d; return nil }
}

// OptionBind sets the Bind in the options struct. If this is not specified [DefaultBind] will be used.
func OptionBind(b Bind) option {
	if b == nil {
		return OptionNop()
	}
	return func(o *options) error { o.bind = b; return nil }
}

// OptionLogger adds a logger to the options struct.
func OptionLogger(l *wglog.Logger) option {
	if l == nil {
		return OptionNop()
	}
	return func(o *options) error { o.loggers = append(o.loggers, l); return nil }
}

// OptionConfig specifies a wireguard config to load before the interface is brought up.
func OptionConfig(cfg wgapi.Configurable) option {
	if cfg == nil {
		return OptionNop()
	}
	return func(o *options) error { o.cfg = &cfg; return nil }
}

// OptionNetDevice initializes a userspace networking stack.
// Note: The pointer *p becomes valid and usable only if the [New] function successfully
// completes without returning an error. In case of errors, *p should not be considered reliable.
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

// OptionCloser adds a closer function to the [options] struct.
// Closer functions are called to gracefully close resources when needed.
func OptionCloser(closer func() error) option {
	return func(o *options) error {
		o.closer = append(o.closer, closer)
		return nil
	}
}
