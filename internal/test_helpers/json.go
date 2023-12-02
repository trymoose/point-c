package test_helpers

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func JSONMarshal[O []byte | string](t testing.TB, a any) O {
	t.Helper()
	b, err := json.Marshal(a)
	require.NoError(t, err, "json.Marshal()")
	return O(b)
}
