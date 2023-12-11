package configvalues

import (
	"encoding"
	"encoding/json"
	"github.com/caddyserver/caddy/v2"
	"github.com/tidwall/gjson"
	"strconv"
	"sync"
)

type (
	// Value is an interface for types that can unmarshal text and return a value.
	Value[V any] interface {
		encoding.TextUnmarshaler
		Value() V
	}

	valueConstraint[V, T any] interface {
		*T
		Value[V]
	}

	// CaddyTextUnmarshaler is a generic struct for unmarshaling text into a value.
	// It stores the unmarshaled value and the original text representation.
	CaddyTextUnmarshaler[V, T any, TP valueConstraint[V, T]] struct {
		value    T
		original string
	}
)

// MarshalText marshals the CaddyTextUnmarshaler back to text.
// It returns the original text representation.
func (c CaddyTextUnmarshaler[V, T, TP]) MarshalText() (text []byte, err error) {
	return []byte(c.original), nil
}

var caddyReplacer = sync.OnceValue(caddy.NewReplacer)

// UnmarshalText unmarshals text into the [CaddyTextUnmarshaler]'s value.
// It uses Caddy's replacer for variable expansion in the text before unmarshaling.
func (c *CaddyTextUnmarshaler[V, T, TP]) UnmarshalText(text []byte) error {
	c.original = string(text)
	text = []byte(caddyReplacer().ReplaceAll(c.original, ""))
	return any(&c.value).(encoding.TextUnmarshaler).UnmarshalText(text)
}

// MarshalJSON marshals the [CaddyTextUnmarshaler] into JSON.
// It quotes the text if it's not valid JSON.
func (c *CaddyTextUnmarshaler[V, T, TP]) MarshalJSON() (text []byte, err error) {
	text, err = c.MarshalText()
	if !gjson.ValidBytes(text) {
		text = []byte(strconv.Quote(string(text)))
	}
	return
}

// UnmarshalJSON unmarshals JSON into the [CaddyTextUnmarshaler]'s value.
// It handles JSON strings and unmarshals them as text.
func (c *CaddyTextUnmarshaler[V, T, TP]) UnmarshalJSON(text []byte) error {
	if s := ""; json.Unmarshal(text, &s) == nil {
		text = []byte(s)
	}
	return c.UnmarshalText(text)
}

// Value returns the underlying value of the [CaddyTextUnmarshaler].
func (c *CaddyTextUnmarshaler[V, T, TP]) Value() V {
	return any(&c.value).(Value[V]).Value()
}
