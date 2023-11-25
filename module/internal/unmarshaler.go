package internal

import (
	"encoding/json"
	"fmt"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
)

func Unmarshaler[T any, TP interface {
	*T
	caddyfile.Unmarshaler
}](name string) func(d *caddyfile.Dispenser, resume any) (any, error) {
	return func(d *caddyfile.Dispenser, resume any) (any, error) {
		var v T
		if resume != nil {
			j, ok := resume.(*httpcaddyfile.App)
			if !ok {
				return nil, fmt.Errorf("not a %T", j)
			} else if j.Name != name {
				return nil, fmt.Errorf("expected app with name %q, got %q", name, j.Name)
			}

			if err := json.Unmarshal(j.Value, &v); err != nil {
				return nil, err
			}
		}

		if err := any(&v).(caddyfile.Unmarshaler).UnmarshalCaddyfile(d); err != nil {
			return nil, err
		}

		return &httpcaddyfile.App{Name: name, Value: caddyconfig.JSON(&v, nil)}, nil
	}
}
