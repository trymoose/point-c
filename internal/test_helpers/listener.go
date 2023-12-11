package test_helpers

import (
	"context"
	"errors"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/google/uuid"
	"net"
	"sync"
	"testing"
	"time"
)

func init() {
	caddy.RegisterModule(new(TestListener))
}

var listenerModName = sync.OnceValue(uuid.New().String)

func ListenerModName() string { return listenerModName() }

func (e *TestListener) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  caddy.ModuleID("caddy.listeners.multi.listeners." + ListenerModName()),
		New: func() caddy.Module { return new(TestListener) },
	}
}

type TestListener struct {
	accept      chan net.Conn
	closeAccept func()
	ctx         context.Context
	t           testing.TB

	Called struct {
		Provision          bool
		Cleanup            bool
		Close              bool
		Accept             bool
		Addr               bool
		UnmarshalCaddyfile bool
	} `json:"-"`

	ID   uuid.UUID `json:"id"`
	fail struct {
		Provision          *string
		Cleanup            *string
		UnmarshalCaddyfile *string
	}
}

func (e *TestListener) FailProvision(msg string)          { e.fail.Provision = &msg }
func (e *TestListener) FailCleanup(msg string)            { e.fail.Cleanup = &msg }
func (e *TestListener) FailUnmarshalCaddyfile(msg string) { e.fail.UnmarshalCaddyfile = &msg }

var TestListeners = map[uuid.UUID]*TestListener{}

func NewTestListener(t testing.TB, ctx context.Context) (*TestListener, func()) {
	tc := TestListener{
		ID:     uuid.New(),
		accept: make(chan net.Conn),
		t:      t,
		ctx:    ctx,
	}
	tc.closeAccept = sync.OnceFunc(func() { close(tc.accept) })
	TestListeners[tc.ID] = &tc
	return &tc, func() {
		t.Helper()
		time.AfterFunc(time.Minute, func() { t.Helper(); delete(TestListeners, tc.ID) })
	}
}

func NewTestListeners(t testing.TB, ctx context.Context, n int) ([]*TestListener, func()) {
	t.Helper()
	cl := make([]func(), n)
	ln := make([]*TestListener, n)
	for i := 0; i < n; i++ {
		ln[i], cl[i] = NewTestListener(t, ctx)
	}
	return ln, func() {
		for _, cl := range cl {
			cl()
		}
	}
}

func (e *TestListener) AcceptConn(c net.Conn) {
	e.t.Helper()
	select {
	case <-e.ctx.Done():
	case e.accept <- c:
	}
}

func (e *TestListener) GetReal() *TestListener {
	lns := TestListeners
	e = lns[e.ID]
	e.t.Helper()
	return e
}

func (e *TestListener) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if !d.NextArg() {
			return d.ArgErr()
		} else if err := e.ID.UnmarshalText([]byte(d.Val())); err != nil {
			return err
		}
	}

	e = e.GetReal()
	e.t.Helper()
	e.Called.UnmarshalCaddyfile = true
	if e.fail.UnmarshalCaddyfile != nil {
		return errors.New(*e.fail.UnmarshalCaddyfile)
	}
	return nil
}

func (e *TestListener) Cleanup() error {
	e = e.GetReal()
	e.t.Helper()
	e.Called.Cleanup = true
	if e.fail.Cleanup != nil {
		return errors.New(*e.fail.Cleanup)
	}
	return nil
}

func (e *TestListener) Addr() net.Addr {
	e = e.GetReal()
	e.t.Helper()
	e.Called.Addr = true
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1}
}

func (e *TestListener) Accept() (net.Conn, error) {
	e = e.GetReal()
	e.t.Helper()
	if c, ok := <-e.accept; ok {
		return c, nil
	}
	return nil, net.ErrClosed
}

func (e *TestListener) Close() error { e = e.GetReal(); e.t.Helper(); e.closeAccept(); return nil }

func (e *TestListener) Provision(caddy.Context) error {
	e = e.GetReal()
	e.t.Helper()
	e.Called.Provision = true
	if e.fail.Provision != nil {
		return errors.New(*e.fail.Provision)
	}
	return nil
}
