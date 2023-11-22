package point_c

import (
	"encoding/json"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMultiWrapper_UnmarshalCaddyfile(t *testing.T) {
	tests := []struct {
		name      string
		caddyfile string
		json      string
		wantErr   bool
	}{
		{
			name: "basic",
			caddyfile: `multi {
	point-c foobar 80
	point-c barfoo 443
}`,
			json: `{"listener": [{"name": "foobar", "port": 80}, {"name": "barfoo", "port": 443}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pc MultiWrapper
			d := caddyfile.NewTestDispenser(tt.caddyfile)
			if err := pc.UnmarshalCaddyfile(d); tt.wantErr {
				require.Errorf(t, err, "UnmarshalCaddyfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var pj MultiWrapper
			if err := json.Unmarshal([]byte(tt.json), &pj); err != nil {
				t.Errorf("json.Unmarshal() error = %v", err)
				return
			}

			pcb, err := json.Marshal(pc)
			if err != nil {
				t.Errorf("json.Marshal() error = %v", err)
				return
			}

			pjb, err := json.Marshal(pj)
			if err != nil {
				t.Errorf("json.Marshal() error = %v", err)
				return
			}

			require.JSONEq(t, string(pcb), string(pjb), "caddyfile != json")
			//if !reflect.DeepEqual(pc, pj) {
			//	t.Errorf("\ncaddyfile(%+v) != \njson     (%+v)", pc, pj)
			//}
		})
	}
}
