package configvalues

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/constraints"
	"net"
	"testing"
)

func TestValueBool(t *testing.T) {
	t.Run("true values", func(t *testing.T) {
		testParseBools(t, []string{"1", "t", "T", "TRUE", "true", "True"}, true)
	})

	t.Run("false values", func(t *testing.T) {
		testParseBools(t, []string{"0", "f", "F", "FALSE", "false", "False"}, false)
	})

	t.Run("invalid", func(t *testing.T) {
		require.Error(t, testParseBool(t, "", false))
	})
}

func testParseBools(t testing.TB, b []string, expected bool) {
	t.Helper()
	for _, b := range b {
		require.NoError(t, testParseBool(t, b, expected))
	}
}

func testParseBool(t testing.TB, b string, expected bool) error {
	t.Helper()
	var vb ValueBool
	if err := vb.UnmarshalText([]byte(b)); err != nil {
		return err
	}
	require.Exactly(t, expected, vb.Value())
	return nil
}

func TestValueString(t *testing.T) {
	var vs ValueString
	const testStr = "foobar"
	require.NoError(t, vs.UnmarshalText([]byte(testStr)))
	require.Exactly(t, testStr, vs.Value())
}

func TestValueUnsigned(t *testing.T) {
	testValueUnsigned[uint](t)
	testValueUnsigned[uint8](t)
	testValueUnsigned[uint16](t)
	testValueUnsigned[uint32](t)
	testValueUnsigned[uint64](t)
	testValueUnsigned[uintptr](t)
}

func testValueUnsigned[N constraints.Unsigned](t *testing.T) {
	t.Helper()
	testValueUnsignedInvalid[N](t, "")
	testValueUnsignedInvalid[N](t, "abc")
	testValueUnsignedInvalid[N](t, "+123")
	testValueUnsignedInvalid[N](t, "-123")
	testValueUnsignedValid[N](t, 0)
	testValueUnsignedValid[N](t, 1)
	testValueUnsignedValid[N](t, 10)
	testValueUnsignedValid[N](t, ^(*new(N)))
}

func testValueUnsignedInvalid[N constraints.Unsigned](t *testing.T, b string) {
	t.Helper()
	t.Run(fmt.Sprintf("%T invalid parse %q", *new(N), b), func(t *testing.T) {
		var vu ValueUnsigned[N]
		require.Error(t, vu.UnmarshalText([]byte(b)))
	})
}

func testValueUnsignedValid[N constraints.Unsigned](t *testing.T, n N) {
	t.Helper()
	t.Run(fmt.Sprintf("%[1]T parse %[1]d", n), func(t *testing.T) {
		var vu ValueUnsigned[N]
		require.NoError(t, vu.UnmarshalText([]byte(fmt.Sprintf("%d", n))))
		require.Exactly(t, vu.Value(), n)
	})
}

func TestValueIP(t *testing.T) {
	t.Run("invalid address", func(t *testing.T) {
		var vip ValueIP
		require.Error(t, vip.UnmarshalText([]byte{'%'}))
	})

	t.Run("ipv4", func(t *testing.T) {
		var vip ValueIP
		v4 := net.IPv4(1, 1, 1, 1)
		b, err := v4.MarshalText()
		require.NoError(t, err)
		require.NoError(t, vip.UnmarshalText(b))
		require.Exactly(t, v4, vip.Value())
	})

	t.Run("ipv6", func(t *testing.T) {
		var vip ValueIP
		v6 := net.ParseIP("abcd:23::33")
		b, err := v6.MarshalText()
		require.NoError(t, err)
		require.NoError(t, vip.UnmarshalText(b))
		require.Exactly(t, v6, vip.Value())
	})
}

func TestValueUDPAddr(t *testing.T) {
	t.Run("invalid address", func(t *testing.T) {
		var vu ValueUDPAddr
		require.Error(t, vu.UnmarshalText([]byte{'.'}))
	})

	t.Run("ipv4", func(t *testing.T) {
		var vu ValueUDPAddr
		addr, err := net.ResolveUDPAddr("udp", "1.1.1.1:0")
		require.NoError(t, err)
		require.NoError(t, vu.UnmarshalText([]byte(addr.String())))
		require.Exactly(t, addr, vu.Value())
	})
}
