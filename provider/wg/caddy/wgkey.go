package caddy

import (
	"encoding"
	"github.com/trymoose/point-c/wgapi"
	"github.com/trymoose/point-c/wgcaddy/pkg/configvalues"
)

type WGKey[K wgapi.PrivateKey | wgapi.PublicKey | wgapi.PresharedKey] struct{ K K }

func (wgk *WGKey[K]) UnmarshalText(text []byte) error {
	return any(&wgk.K).(encoding.TextUnmarshaler).UnmarshalText(text)
}

func (wgk WGKey[K]) Value() K {
	return wgk.K
}

type (
	// PrivateKey is a wireguard private key in base64 format.
	PrivateKey = configvalues.CaddyTextUnmarshaler[wgapi.PrivateKey, caddy.WGKey[wgapi.PrivateKey], *caddy.WGKey[wgapi.PrivateKey]]
	// PublicKey is a wireguard public key in base64 format.
	PublicKey = configvalues.CaddyTextUnmarshaler[wgapi.PublicKey, caddy.WGKey[wgapi.PublicKey], *caddy.WGKey[wgapi.PublicKey]]
	// PresharedKey is a wireguard preshared key in base64 format.
	PresharedKey = configvalues.CaddyTextUnmarshaler[wgapi.PresharedKey, caddy.WGKey[wgapi.PresharedKey], *caddy.WGKey[wgapi.PresharedKey]]
)
