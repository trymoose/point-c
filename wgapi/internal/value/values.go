package value

import (
	"encoding/hex"
	"fmt"
	"github.com/trymoose/wg4d/wgapi/internal/value/wgkey"
	"golang.org/x/exp/constraints"
	"net"
	"strconv"
)

type Uint32[K fmt.Stringer] uint32

func (v Uint32[K]) Key() string    { return getKey[K]() }
func (v Uint32[K]) String() string { return numString(v) }

type Uint16[K fmt.Stringer] uint16

func (v Uint16[K]) Key() string    { return getKey[K]() }
func (v Uint16[K]) String() string { return numString(v) }

type Uint64[K fmt.Stringer] uint64

func (v Uint64[K]) Key() string    { return getKey[K]() }
func (v Uint64[K]) String() string { return numString(v) }

type Int64[K fmt.Stringer] int64

func (v Int64[K]) Key() string    { return getKey[K]() }
func (v Int64[K]) String() string { return numString(v) }

type True[K fmt.Stringer] struct{}

func (True[K]) Key() string    { return getKey[K]() }
func (True[K]) String() string { return boolString(true) }

type One[K fmt.Stringer] struct{}

func (One[K]) Key() string    { return getKey[K]() }
func (One[K]) String() string { return numString(1) }

type UDPAddr[K fmt.Stringer] net.UDPAddr

func (addr UDPAddr[K]) Key() string    { return getKey[K]() }
func (addr UDPAddr[K]) String() string { return ((*net.UDPAddr)(&addr)).String() }

type IPNet[K fmt.Stringer] net.IPNet

func (ipnet IPNet[K]) Key() string    { return getKey[K]() }
func (ipnet IPNet[K]) String() string { return ((*net.IPNet)(&ipnet)).String() }

type Key[K fmt.Stringer, Type wgkey.Type] wgkey.Key[Type]

func (key Key[K, Type]) Key() string    { return getKey[K]() }
func (key Key[K, Type]) String() string { return hex.EncodeToString(key[:]) }

func getKey[K fmt.Stringer]() string              { var k K; return k.String() }
func boolString[B ~bool](b B) string              { return strconv.FormatBool(bool(b)) }
func numString[N constraints.Integer](n N) string { return strconv.Itoa(int(n)) }
