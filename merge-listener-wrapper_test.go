package point_c_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	_ "github.com/caddyserver/caddy/v2/modules/standard"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	pointc "github.com/trymoose/point-c"
	_ "github.com/trymoose/point-c/module"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func init() {
	caddy.RegisterModule(new(TestListener))
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

type NopConn struct{}

func (n2 *NopConn) Read([]byte) (n int, err error)    { return 0, io.EOF }
func (n2 *NopConn) Write(b []byte) (n int, err error) { return len(b), nil }
func (n2 *NopConn) Close() error                      { return nil }
func (n2 *NopConn) LocalAddr() net.Addr               { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1} }
func (n2 *NopConn) RemoteAddr() net.Addr              { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 2} }
func (n2 *NopConn) SetDeadline(time.Time) error       { return nil }
func (n2 *NopConn) SetReadDeadline(time.Time) error   { return nil }
func (n2 *NopConn) SetWriteDeadline(time.Time) error  { return nil }

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

var modName = uuid.New().String()

func (e *TestListener) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  caddy.ModuleID("caddy.listeners.merge.listeners." + modName),
		New: func() caddy.Module { return new(TestListener) },
	}
}

func GenerateJSON(t testing.TB, tln ...*TestListener) []byte {
	t.Helper()
	raw := make([]string, len(tln))
	for i, ln := range tln {
		raw[i] = fmt.Sprintf(`{"listener": %q, "ID": %q}`, modName, ln.ID.String())
	}
	return []byte(fmt.Sprintf(`{"listeners": [%s]}`, strings.Join(raw, ",")))
}

func TestMergeWrapper_WrapListener(t *testing.T) {
	t.Run("closed before accepted", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		ln1, clean := NewTestListener(t, ctx)
		defer clean()
		v, err := ctx.LoadModuleByID("caddy.listeners.merge", GenerateJSON(t, ln1))
		require.NoError(t, err)

		ln2, clean := NewTestListener(t, ctx)
		defer clean()
		wrapped := v.(caddy.ListenerWrapper).WrapListener(ln2)
		ln1.AcceptConn(&NopConn{})
		require.NoError(t, wrapped.Close())
	})

	t.Run("one listener, accept from wrapped", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		ln1, clean := NewTestListener(t, ctx)
		defer clean()
		v, err := ctx.LoadModuleByID("caddy.listeners.merge", GenerateJSON(t, ln1))
		require.NoError(t, err)

		ln2, clean := NewTestListener(t, ctx)
		defer clean()
		wrapped := v.(caddy.ListenerWrapper).WrapListener(ln2)
		conn, errs := make(chan net.Conn), make(chan error)
		go func() {
			defer wrapped.Close()
			c, e := wrapped.Accept()
			if e != nil {
				select {
				case errs <- e:
				case <-ctx.Done():
				}
			} else {
				select {
				case conn <- c:
				case <-ctx.Done():
				}
			}
		}()

		go func() { ln2.AcceptConn(&NopConn{}) }()

		select {
		case err := <-errs:
			require.NoError(t, err)
		case c := <-conn:
			require.NotNil(t, c)
		case <-ctx.Done():
			t.Error("context cancelled")
		case <-time.After(time.Second * 5):
			t.Error("test timed out")
		}
	})

	t.Run("one listener, accept from merged", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		ln1, clean := NewTestListener(t, ctx)
		defer clean()
		defer cancel()
		v, err := ctx.LoadModuleByID("caddy.listeners.merge", GenerateJSON(t, ln1))
		require.NoError(t, err)

		ln2, clean := NewTestListener(t, ctx)
		defer clean()
		wrapped := v.(caddy.ListenerWrapper).WrapListener(ln2)
		conn, errs := make(chan net.Conn), make(chan error)
		go func() {
			defer wrapped.Close()
			c, e := wrapped.Accept()
			if e != nil {
				select {
				case errs <- e:
				case <-ctx.Done():
				}
			} else {
				select {
				case conn <- c:
				case <-ctx.Done():
				}
			}
		}()

		go func() { ln1.AcceptConn(&NopConn{}) }()

		select {
		case err := <-errs:
			require.NoError(t, err)
		case c := <-conn:
			require.NotNil(t, c)
		case <-ctx.Done():
			t.Error("context cancelled")
		case <-time.After(time.Second * 5):
			t.Error("test timed out")
		}
	})

	t.Run("two listeners, accept from wrapped", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()

		ln1, clean := NewTestListener(t, ctx)
		defer clean()
		ln2, clean := NewTestListener(t, ctx)
		defer clean()
		v, err := ctx.LoadModuleByID("caddy.listeners.merge", GenerateJSON(t, ln1, ln2))
		require.NoError(t, err)

		ln3, clean := NewTestListener(t, ctx)
		defer clean()
		wrapped := v.(caddy.ListenerWrapper).WrapListener(ln3)
		conn, errs := make(chan net.Conn), make(chan error)
		go func() {
			defer wrapped.Close()
			c, e := wrapped.Accept()
			if e != nil {
				select {
				case errs <- e:
				case <-ctx.Done():
				}
			} else {
				select {
				case conn <- c:
				case <-ctx.Done():
				}
			}
		}()

		go func() { ln3.AcceptConn(&NopConn{}) }()

		select {
		case err := <-errs:
			require.NoError(t, err)
		case c := <-conn:
			require.NotNil(t, c)
		case <-ctx.Done():
			t.Error("context cancelled")
		case <-time.After(time.Second * 5):
			t.Error("test timed out")
		}
	})

	t.Run("two listeners, accept from merged 1", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()

		ln1, clean := NewTestListener(t, ctx)
		defer clean()
		ln2, clean := NewTestListener(t, ctx)
		defer clean()
		v, err := ctx.LoadModuleByID("caddy.listeners.merge", GenerateJSON(t, ln1, ln2))
		require.NoError(t, err)

		ln3, clean := NewTestListener(t, ctx)
		defer clean()
		wrapped := v.(caddy.ListenerWrapper).WrapListener(ln3)
		conn, errs := make(chan net.Conn), make(chan error)
		go func() {
			defer wrapped.Close()
			c, e := wrapped.Accept()
			if e != nil {
				select {
				case errs <- e:
				case <-ctx.Done():
				}
			} else {
				select {
				case conn <- c:
				case <-ctx.Done():
				}
			}
		}()

		go func() { ln1.AcceptConn(&NopConn{}) }()

		select {
		case err := <-errs:
			require.NoError(t, err)
		case c := <-conn:
			require.NotNil(t, c)
		case <-ctx.Done():
			t.Error("context cancelled")
		case <-time.After(time.Second * 5):
			t.Error("test timed out")
		}
	})

	t.Run("two listeners, accept from merged 1", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()

		ln1, clean := NewTestListener(t, ctx)
		defer clean()
		ln2, clean := NewTestListener(t, ctx)
		defer clean()
		v, err := ctx.LoadModuleByID("caddy.listeners.merge", GenerateJSON(t, ln1, ln2))
		require.NoError(t, err)

		ln3, clean := NewTestListener(t, ctx)
		defer clean()
		wrapped := v.(caddy.ListenerWrapper).WrapListener(ln3)
		conn, errs := make(chan net.Conn), make(chan error)
		go func() {
			defer wrapped.Close()
			c, e := wrapped.Accept()
			if e != nil {
				select {
				case errs <- e:
				case <-ctx.Done():
				}
			} else {
				select {
				case conn <- c:
				case <-ctx.Done():
				}
			}
		}()

		go func() { ln2.AcceptConn(&NopConn{}) }()

		select {
		case err := <-errs:
			require.NoError(t, err)
		case c := <-conn:
			require.NotNil(t, c)
		case <-ctx.Done():
			t.Error("context cancelled")
		case <-time.After(time.Second * 5):
			t.Error("test timed out")
		}
	})
}

func TestMergeWrapper_Cleanup(t *testing.T) {
	t.Run("listener closed", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		ln, clean := NewTestListener(t, ctx)
		defer clean()
		v, err := ctx.LoadModuleByID("caddy.listeners.merge", GenerateJSON(t, ln))
		require.NoError(t, err)
		cancel()
		require.Exactly(t, pointc.MergeWrapper{}, *v.(*pointc.MergeWrapper))
	})
}

func TestMergeWrapper_Provision(t *testing.T) {
	t.Run("no listeners given", func(t *testing.T) {
		var v pointc.MergeWrapper
		require.Error(t, v.Provision(caddy.Context{}))
	})

	t.Run("listeners set to null", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		_, err := ctx.LoadModuleByID("caddy.listeners.merge", []byte(`{"listeners": null}`))
		require.Error(t, err)
	})

	t.Run("listener failed to load", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		ln, clean := NewTestListener(t, ctx)
		defer clean()
		defer cancel()
		ln.FailProvision(ln.ID.String())
		_, err := ctx.LoadModuleByID("caddy.listeners.merge", GenerateJSON(t, ln))
		require.ErrorContains(t, err, ln.ID.String())
	})

	t.Run("listener fully provisions one listeners", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		ln, clean := NewTestListener(t, ctx)
		defer clean()
		defer cancel()
		_, err := ctx.LoadModuleByID("caddy.listeners.merge", GenerateJSON(t, ln))
		require.NoError(t, err)
	})

	t.Run("listener fully provisions two listeners", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		ln1, clean := NewTestListener(t, ctx)
		defer clean()
		ln2, clean := NewTestListener(t, ctx)
		defer clean()
		defer cancel()
		_, err := ctx.LoadModuleByID("caddy.listeners.merge", GenerateJSON(t, ln1, ln2))
		require.NoError(t, err)
	})
}

func TestMergeWrapper_UnmarshalCaddyfile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	ln1, clean := NewTestListener(t, ctx)
	defer clean()
	ln2, clean := NewTestListener(t, ctx)
	defer clean()

	tests := []struct {
		name      string
		caddyfile string
		json      string
		wantErr   bool
	}{
		{
			name: "basic",
			caddyfile: `multi {
	` + modName + ` ` + ln1.ID.String() + `
	` + modName + ` ` + ln2.ID.String() + `
}`,
			json: `{"listeners": [{"id": "` + ln1.ID.String() + `"}, {"id": "` + ln2.ID.String() + `"}]}`,
		},
		{
			name: "submodule does not exist",
			caddyfile: `multi {
	foobar
}`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pc pointc.MergeWrapper
			if err := pc.UnmarshalCaddyfile(caddyfile.NewTestDispenser(tt.caddyfile)); tt.wantErr {
				require.Errorf(t, err, "UnmarshalCaddyfile() wantErr %v", tt.wantErr)
				return
			} else {
				require.NoError(t, err, "UnmarshalCaddyfile()")
			}
			require.JSONEq(t, tt.json, jsonMarshal[string](t, pc), "caddyfile != json")
		})
	}
}

func jsonMarshal[O []byte | string](t testing.TB, a any) O {
	t.Helper()
	b, err := json.Marshal(a)
	require.NoError(t, err, "json.Marshal()")
	return O(b)
}
