package point_c_test

import (
	"context"
	"fmt"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	_ "github.com/caddyserver/caddy/v2/modules/standard"
	"github.com/stretchr/testify/require"
	pointc "github.com/trymoose/point-c"
	"github.com/trymoose/point-c/internal/test_helpers"
	_ "github.com/trymoose/point-c/module"
	"golang.org/x/exp/rand"
	"net"
	"strings"
	"testing"
	"time"
)

func TestMergeWrapper_WrapListener(t *testing.T) {
	t.Run("closed before accepted", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		ln, clean := test_helpers.NewTestListeners(t, ctx, 2)
		defer clean()
		ln1, ln2 := ln[0], ln[1]
		v, err := ctx.LoadModuleByID("caddy.listeners.merge", generateMergedJSON(t, ln1))
		require.NoError(t, err)

		wrapped := v.(caddy.ListenerWrapper).WrapListener(ln2)
		ln1.AcceptConn(test_helpers.NopConn())
		require.NoError(t, wrapped.Close())
	})

	t.Run("one listener, accept from wrapped", func(t *testing.T) {
		acceptTest(t, 1, func(t testing.TB, wrapped *test_helpers.TestListener, _ []*test_helpers.TestListener) []*test_helpers.TestListener {
			t.Helper()
			return []*test_helpers.TestListener{wrapped}
		})
	})

	t.Run("one listener, accept from merged", func(t *testing.T) {
		acceptTest(t, 1, func(t testing.TB, _ *test_helpers.TestListener, lns []*test_helpers.TestListener) []*test_helpers.TestListener {
			t.Helper()
			return lns
		})
	})

	t.Run("two listeners, accept from wrapped", func(t *testing.T) {
		acceptTest(t, 1, func(t testing.TB, wrapped *test_helpers.TestListener, _ []*test_helpers.TestListener) []*test_helpers.TestListener {
			t.Helper()
			return []*test_helpers.TestListener{wrapped}
		})
	})

	t.Run("two listeners, accept from one random merged", func(t *testing.T) {
		acceptTest(t, 1, func(t testing.TB, _ *test_helpers.TestListener, lns []*test_helpers.TestListener) []*test_helpers.TestListener {
			t.Helper()
			return []*test_helpers.TestListener{lns[rand.Intn(len(lns))]}
		})
	})

	t.Run("two listeners, accept from both", func(t *testing.T) {
		acceptTest(t, 1, func(t testing.TB, _ *test_helpers.TestListener, lns []*test_helpers.TestListener) []*test_helpers.TestListener {
			t.Helper()
			return lns
		})
	})

	t.Run("three listeners, accept all", func(t *testing.T) {
		acceptTest(t, 1, func(t testing.TB, wrapped *test_helpers.TestListener, lns []*test_helpers.TestListener) []*test_helpers.TestListener {
			t.Helper()
			lns = append([]*test_helpers.TestListener{wrapped}, lns...)
			rand.Shuffle(len(lns), func(i, j int) { lns[i], lns[j] = lns[j], lns[i] })
			return lns
		})
	})

	t.Run("three listeners, accept wrapped and one merged", func(t *testing.T) {
		acceptTest(t, 1, func(t testing.TB, wrapped *test_helpers.TestListener, lns []*test_helpers.TestListener) []*test_helpers.TestListener {
			t.Helper()
			return []*test_helpers.TestListener{wrapped, lns[rand.Intn(len(lns))]}
		})
	})
}

func acceptTest(t testing.TB, n int, acceptor func(t testing.TB, wrapped *test_helpers.TestListener, lns []*test_helpers.TestListener) []*test_helpers.TestListener) {
	t.Helper()
	n = max(n, 1)
	ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
	defer cancel()

	ln, clean := test_helpers.NewTestListeners(t, ctx, n+1) // create an extra one to be wrapped
	defer clean()
	v, err := ctx.LoadModuleByID("caddy.listeners.merge", generateMergedJSON(t, ln[1:]...))
	require.NoError(t, err)

	wrapped := v.(caddy.ListenerWrapper).WrapListener(ln[0])
	conn, errs := make(chan net.Conn), make(chan error)
	accept := acceptor(t, ln[0], ln[1:])
	go func() {
		defer wrapped.Close()
		for range accept {
			c, e := wrapped.Accept()
			if e != nil {
				select {
				case errs <- e:
				case <-ctx.Done():
					return
				}
			} else {
				select {
				case conn <- c:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	go func() {
		for _, a := range accept {
			select {
			case <-ctx.Done():
				return
			default:
				a.AcceptConn(test_helpers.NopConn())
			}
		}
	}()

	for i := range accept {
		select {
		case err := <-errs:
			require.NoError(t, err, "i = %d", i)
		case c := <-conn:
			require.NotNil(t, c, "i = %d", i)
		case <-ctx.Done():
			t.Error("context cancelled", "i = %d", i)
		case <-time.After(time.Second * 5):
			t.Error("test timed out", "i = %d", i)
		}
	}
}

func TestMergeWrapper_Cleanup(t *testing.T) {
	t.Run("listener closed", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		ln, clean := test_helpers.NewTestListener(t, ctx)
		defer clean()
		v, err := ctx.LoadModuleByID("caddy.listeners.merge", generateMergedJSON(t, ln))
		require.NoError(t, err)
		cancel()
		require.Exactly(t, pointc.MergeWrapper{}, *v.(*pointc.MergeWrapper))
	})
}

func TestMergeWrapper_Provision(t *testing.T) {
	t.Run("no listeners given", func(t *testing.T) {
		var v pointc.MergeWrapper
		require.Error(t, v.Provision(caddy.Context{}))
	})

	t.Run("listeners set to null", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		_, err := ctx.LoadModuleByID("caddy.listeners.merge", []byte(`{"listeners": null}`))
		require.Error(t, err)
	})

	t.Run("listener failed to load", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		ln, clean := test_helpers.NewTestListener(t, ctx)
		defer clean()
		ln.FailProvision(ln.ID.String())
		_, err := ctx.LoadModuleByID("caddy.listeners.merge", generateMergedJSON(t, ln))
		require.ErrorContains(t, err, ln.ID.String())
	})

	t.Run("listener fully provisions one listeners", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		ln, clean := test_helpers.NewTestListener(t, ctx)
		defer clean()
		_, err := ctx.LoadModuleByID("caddy.listeners.merge", generateMergedJSON(t, ln))
		require.NoError(t, err)
	})

	t.Run("listener fully provisions two listeners", func(t *testing.T) {
		ctx, cancel := caddy.NewContext(caddy.Context{Context: context.TODO()})
		defer cancel()
		ln, clean := test_helpers.NewTestListeners(t, ctx, 2)
		defer clean()
		_, err := ctx.LoadModuleByID("caddy.listeners.merge", generateMergedJSON(t, ln...))
		require.NoError(t, err)
	})
}

func TestMergeWrapper_UnmarshalCaddyfile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	ln, clean := test_helpers.NewTestListeners(t, ctx, 2)
	defer clean()
	ln1, ln2 := ln[0], ln[1]

	tests := []struct {
		name      string
		caddyfile string
		json      string
		wantErr   bool
	}{
		{
			name: "basic",
			caddyfile: `multi {
	` + test_helpers.ListenerModName() + ` ` + ln1.ID.String() + `
	` + test_helpers.ListenerModName() + ` ` + ln2.ID.String() + `
}`,
			json: `{"listeners": [{"id": "` + ln1.ID.String() + `"}, {"id": "` + ln2.ID.String() + `"}]}`,
		},
		{
			name: "submodule does not exist",
			caddyfile: `multi {
	foobar
}`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pc pointc.MergeWrapper
			if err := pc.UnmarshalCaddyfile(caddyfile.NewTestDispenser(tt.caddyfile)); tt.wantErr {
				require.Errorf(t, err, "UnmarshalCaddyfile() wantErr %v", tt.wantErr)
				return
			} else {
				require.NoError(t, err, "UnmarshalCaddyfile()")
			}
			require.JSONEq(t, tt.json, test_helpers.JSONMarshal[string](t, pc), "caddyfile != json")
		})
	}
}

func generateMergedJSON(t testing.TB, tln ...*test_helpers.TestListener) []byte {
	t.Helper()
	raw := make([]string, len(tln))
	for i, ln := range tln {
		raw[i] = fmt.Sprintf(`{"listener": %q, "ID": %q}`, test_helpers.ListenerModName(), ln.ID.String())
	}
	return []byte(fmt.Sprintf(`{"listeners": [%s]}`, strings.Join(raw, ",")))
}
