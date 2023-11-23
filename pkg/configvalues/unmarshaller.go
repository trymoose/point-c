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
	Value[V any] interface {
		encoding.TextUnmarshaler
		Value() V
	}
	valueConstraint[V, T any] interface {
		*T
		Value[V]
	}
	CaddyTextUnmarshaler[V, T any, TP valueConstraint[V, T]] struct {
		value    T
		original string
	}
)

func NewCaddyTextUnmarshaler[V, T any, TP valueConstraint[V, T]](text string) (*CaddyTextUnmarshaler[V, T, TP], error) {
	var c CaddyTextUnmarshaler[V, T, TP]
	if err := c.UnmarshalText([]byte(text)); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c CaddyTextUnmarshaler[V, T, TP]) MarshalText() (text []byte, err error) {
	return []byte(c.original), nil
}

var caddyReplacer = sync.OnceValue(caddy.NewReplacer)

func (c *CaddyTextUnmarshaler[V, T, TP]) UnmarshalText(text []byte) error {
	c.original = string(text)
	text = []byte(caddyReplacer().ReplaceAll(c.original, ""))
	return any(&c.value).(encoding.TextUnmarshaler).UnmarshalText(text)
}

func (c *CaddyTextUnmarshaler[V, T, TP]) MarshalJSON() (text []byte, err error) {
	text, err = c.MarshalText()
	if !gjson.ValidBytes(text) {
		text = []byte(strconv.Quote(string(text)))
	}
	return
}

func (c *CaddyTextUnmarshaler[V, T, TP]) UnmarshalJSON(text []byte) error {
	if s := ""; json.Unmarshal(text, &s) == nil {
		text = []byte(s)
	}
	return c.UnmarshalText(text)
}

func (c *CaddyTextUnmarshaler[V, T, TP]) Value() V {
	return any(&c.value).(Value[V]).Value()
}
