package point_c_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	pointc "github.com/trymoose/point-c"
	"github.com/trymoose/point-c/internal/test_helpers"
	"testing"
)

func (t *TestNetwork) CaddyModule() caddy.ModuleInfo {
	t.t.Helper()
	return caddy.ModuleInfo{
		ID:  t.ID(),
		New: func() caddy.Module { t.t.Helper(); return t },
	}
}

func (t *TestNetwork) ID() caddy.ModuleID {
	t.t.Helper()
	return caddy.ModuleID("point-c.net.test-%s" + t.id.String())
}

type TestNetwork struct {
	t  testing.TB
	id uuid.UUID
}

func (t *TestNetwork) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	return nil
}

func NewTestNet(t testing.TB) *TestNetwork {
	t.Helper()
	tn := &TestNetwork{
		t:  t,
		id: uuid.New(),
	}
	return tn
}

func (t *TestNetwork) Networks() map[string]pointc.Net {
	return map[string]pointc.Net{}
}

func TestPointc_StartStop(t *testing.T) {
	ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
	defer cancel()
	v, err := ctx.LoadModuleByID("point-c", json.RawMessage(`{}`))
	require.NoError(t, err)
	app, ok := v.(caddy.App)
	require.True(t, ok)
	require.NoError(t, app.Start())
	require.NoError(t, app.Stop())
}

func TestPointc_Lookup(t *testing.T) {
	t.Run("not exists", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		v, err := ctx.LoadModuleByID("point-c", json.RawMessage(`{}`))
		require.NoError(t, err)
		lookup, ok := v.(pointc.NetLookup)
		require.True(t, ok)
		n, ok := lookup.Lookup("")
		require.False(t, ok)
		require.Nil(t, n)
	})

	t.Run("exists", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		v, err := ctx.LoadModuleByID("point-c", json.RawMessage(`{}`))
		require.NoError(t, err)
		lookup, ok := v.(pointc.NetLookup)
		require.True(t, ok)
		n, ok := lookup.Lookup("")
		require.False(t, ok)
		require.Nil(t, n)
	})
}

func TestPointc_Provision(t *testing.T) {
	t.Run("null networks", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		_, err := ctx.LoadModuleByID("point-c", json.RawMessage(`{}`))
		require.NoError(t, err)
	})

	t.Run("empty network slice networks", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		_, err := ctx.LoadModuleByID("point-c", json.RawMessage(`{"networks": []}`))
		require.NoError(t, err)
	})
}

func TestPointc_UnmarshalCaddyfile(t *testing.T) {
	testNet := NewTestNet(t)
	caddy.RegisterModule(testNet)

	tests := []struct {
		name      string
		caddyfile string
		json      string
		wantErr   bool
	}{
		{
			name: "basic",
			caddyfile: fmt.Sprintf(`point-c {
	%[1]s
	%[1]s
}`, testNet.ID().Name()),
			json: `{"networks": [{}, {}]}`,
		},
		{
			name: "submodule does not exist",
			caddyfile: `point-c {
	foobar
}`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pc pointc.Pointc
			if err := pc.UnmarshalCaddyfile(caddyfile.NewTestDispenser(tt.caddyfile)); tt.wantErr {
				require.Errorf(t, err, "UnmarshalCaddyfile() wantErr %v", tt.wantErr)
				return
			} else {
				require.NoError(t, err, "UnmarshalCaddyfile()")
			}
			require.JSONEq(t, test_helpers.JSONMarshal[string](t, pc), tt.json, "caddyfile != json")
		})
	}
}
