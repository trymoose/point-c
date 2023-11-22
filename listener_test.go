package point_c

import (
	"encoding/json"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"reflect"
	"testing"
)

func TestListener_UnmarshalCaddyfile(t *testing.T) {
	tests := []struct {
		name      string
		caddyfile string
		json      string
		wantErr   bool
	}{
		{
			name:      "basic",
			caddyfile: "point-c remote 80",
			json:      `{"name": "remote", "port": 80}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pc Listener
			d := caddyfile.NewTestDispenser(tt.caddyfile)
			if err := pc.UnmarshalCaddyfile(d); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalCaddyfile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var pj Listener
			if err := json.Unmarshal([]byte(tt.json), &pj); err != nil {
				t.Errorf("json.Unmarshal() error = %v", err)
				return
			}

			if !reflect.DeepEqual(pc, pj) {
				t.Errorf("caddyfile(%+v) != json(%+v)", pc, pj)
			}
		})
	}
}
