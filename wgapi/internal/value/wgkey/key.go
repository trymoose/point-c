package wgkey

import (
	"fmt"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"slices"
)

type (
	Key[T Type] wgtypes.Key
	Type        interface{ keyType() }
	Private     struct{}
	Public      struct{}
	PreShared   struct{}
)

func (Public) keyType()    {}
func (Private) keyType()   {}
func (PreShared) keyType() {}

func (k *Key[T]) MarshalText() ([]byte, error) {
	return []byte(wgtypes.Key(*k).String()), nil
}

func (k *Key[T]) UnmarshalText(text []byte) error {
	key, err := wgtypes.ParseKey(string(text))
	if err != nil {
		return err
	}
	copy(k[:], key[:])
	return nil
}

func (k *Key[Type]) Public() (public *Key[Public], err error) {
	switch v := any(k).(type) {
	case *Key[Public]:
		public = (*Key[Public])(slices.Clone(k[:]))
	case *Key[Private]:
		k := Key[Public](wgtypes.Key(*k).PublicKey())
		public = &k
	default:
		err = fmt.Errorf("invalid key type %T", v)
	}
	return
}
