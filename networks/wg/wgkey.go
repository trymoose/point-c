package wg

import (
	"encoding"
	"github.com/trymoose/point-c/pkg/configvalues"
	"github.com/trymoose/point-c/pkg/wg/wgapi"
)

type valueKey[K wgapi.PrivateKey | wgapi.PublicKey | wgapi.PresharedKey] struct{ K K }

func (wgk *valueKey[K]) UnmarshalText(text []byte) error {
	return any(&wgk.K).(encoding.TextUnmarshaler).UnmarshalText(text)
}

func (wgk valueKey[K]) Value() K {
	return wgk.K
}

type (
	// PrivateKey is a wireguard private key in base64 format.
	PrivateKey = configvalues.CaddyTextUnmarshaler[wgapi.PrivateKey, valueKey[wgapi.PrivateKey], *valueKey[wgapi.PrivateKey]]
	// PublicKey is a wireguard public key in base64 format.
	PublicKey = configvalues.CaddyTextUnmarshaler[wgapi.PublicKey, valueKey[wgapi.PublicKey], *valueKey[wgapi.PublicKey]]
	// PresharedKey is a wireguard preshared key in base64 format.
	PresharedKey = configvalues.CaddyTextUnmarshaler[wgapi.PresharedKey, valueKey[wgapi.PresharedKey], *valueKey[wgapi.PresharedKey]]
)
