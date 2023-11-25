package point_c_test

import (
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	pointc "github.com/trymoose/point-c"
	"testing"
)

type TestNet struct {
	t  testing.TB
	id uuid.UUID
}

func (t *TestNet) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	return nil
}

func NewTestNet(t testing.TB) *TestNet {
	t.Helper()
	return &TestNet{
		t:  t,
		id: uuid.New(),
	}
}

func (t *TestNet) ID() caddy.ModuleID {
	t.t.Helper()
	return caddy.ModuleID("point-c.net.test-%s" + t.id.String())
}

func (t *TestNet) CaddyModule() caddy.ModuleInfo {
	t.t.Helper()
	return caddy.ModuleInfo{
		ID:  t.ID(),
		New: func() caddy.Module { t.t.Helper(); return t },
	}
}

func (t *TestNet) Networks() map[string]pointc.Net {
	return map[string]pointc.Net{}
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
			require.JSONEq(t, jsonMarshal[string](t, pc), tt.json, "caddyfile != json")
		})
	}
}
