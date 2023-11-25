package point_c_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	_ "github.com/caddyserver/caddy/v2/modules/standard"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	pointc "github.com/trymoose/point-c"
	_ "github.com/trymoose/point-c/module"
	channel_listener "github.com/trymoose/point-c/pkg/channel-listener"
	"net"
	"strings"
	"sync/atomic"
	"testing"
)

func init() {
	caddy.RegisterModule(new(TestConn))
}

type TestConn struct {
	cln            atomic.Pointer[channel_listener.Listener]
	accept         chan net.Conn
	ID             uuid.UUID
	ProvisionError *string `json:",omitempty"`
}

var TestConns = map[uuid.UUID]*TestConn{}

func GetConn(t testing.TB, id uuid.UUID) *TestConn {
	t.Helper()
	v, ok := TestConns[id]
	require.Exactly(t, true, ok)
	delete(TestConns, id)
	return v
}

func (e *TestConn) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if !d.NextArg() {
			return d.ArgErr()
		} else if err := e.ID.UnmarshalText([]byte(d.Val())); err != nil {
			return err
		}
	}
	return nil
}

func (e *TestConn) Cleanup() error {
	if v := e.cln.Load(); !e.cln.CompareAndSwap(v, nil) {
		close(e.accept)
		v.Close()
		return nil
	}
	return nil
}
func (e *TestConn) Accept() (net.Conn, error) { return e.cln.Load().Accept() }
func (e *TestConn) Close() error              { return e.cln.Load().Close() }
func (e *TestConn) Addr() net.Addr            { return nil }

func (e *TestConn) Provision(caddy.Context) error {
	if e.ProvisionError != nil {
		return errors.New(*e.ProvisionError)
	}
	e.accept = make(chan net.Conn)
	e.cln.Store(channel_listener.New(e.accept, nil))
	TestConns[e.ID] = e
	return nil
}

var modName = uuid.New().String()

func (e *TestConn) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  caddy.ModuleID("caddy.listeners.merge.listeners." + modName),
		New: func() caddy.Module { return new(TestConn) },
	}
}

func TestMergeWrapper_WrapListener(t *testing.T) {
	t.Run("one listeners", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		errId1 := uuid.New()
		v, err := ctx.LoadModuleByID("caddy.listeners.merge", []byte(`{"listeners": [{"listener": "`+modName+`", "ID": "`+errId1.String()+`"}]}`))
		require.NoError(t, err)
		defer v.(caddy.CleanerUpper).Cleanup()
		//v.(caddy.ListenerWrapper).WrapListener()
	})

	t.Run("two listeners", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		errId1, errId2 := uuid.New(), uuid.New()
		v, err := ctx.LoadModuleByID("caddy.listeners.merge", []byte(`{"listeners": [{"listener": "`+modName+`", "ID": "`+errId1.String()+`"}, {"listener": "`+modName+`", "ID": "`+errId2.String()+`"}]}`))
		require.NoError(t, err)
		defer v.(caddy.CleanerUpper).Cleanup()
	})
}

func TestMergeWrapper_ProvisionCleanup(t *testing.T) {
	t.Run("no listeners given", func(t *testing.T) {
		var v pointc.MergeWrapper
		require.Error(t, v.Provision(caddy.Context{}))
	})

	t.Run("listener failed to load", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		errId := uuid.New().String()
		_, err := ctx.LoadModuleByID("caddy.listeners.merge", []byte(`{"listeners": [{"listener": "`+modName+`", "ID": "5b89af9f-1669-4d3e-869d-5dee83ae7cce", "ProvisionError": "`+errId+`"}]}`))
		require.Exactly(t, true, strings.Contains(err.Error(), errId))
	})

	t.Run("listener fully provisions one listeners", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		errId1 := uuid.New()
		v, err := ctx.LoadModuleByID("caddy.listeners.merge", []byte(`{"listeners": [{"listener": "`+modName+`", "ID": "`+errId1.String()+`"}]}`))
		require.NoError(t, err)
		defer v.(caddy.CleanerUpper).Cleanup()
	})

	t.Run("listener fully provisions two listeners", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		errId1, errId2 := uuid.New(), uuid.New()
		v, err := ctx.LoadModuleByID("caddy.listeners.merge", []byte(`{"listeners": [{"listener": "`+modName+`", "ID": "`+errId1.String()+`"}, {"listener": "`+modName+`", "ID": "`+errId2.String()+`"}]}`))
		require.NoError(t, err)
		defer v.(caddy.CleanerUpper).Cleanup()
	})
}

func TestMergeWrapper_UnmarshalCaddyfile(t *testing.T) {
	tests := []struct {
		name      string
		caddyfile string
		json      string
		wantErr   bool
	}{
		{
			name: "basic",
			caddyfile: `multi {
	` + modName + ` 5b89af9f-1669-4d3e-869d-5dee83ae7cce
	` + modName + ` f24527c6-8bbf-4c39-b5fa-148fd2a25309
}`,
			json: `{"listeners": [{"ID": "5b89af9f-1669-4d3e-869d-5dee83ae7cce"}, {"ID": "f24527c6-8bbf-4c39-b5fa-148fd2a25309"}]}`,
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
