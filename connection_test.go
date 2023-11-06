package wg4d_test

import (
	"context"
	"errors"
	"github.com/trymoose/wg4d"
	"github.com/trymoose/wg4d/wgapi"
	"golang.org/x/exp/rand"
	"math"
	"net"
	"testing"
	"time"
)

func TestTCPConnection(t *testing.T) {
	keys := generateKeys(t)
	wgPort := uint16(51820)
	clientPublic := net.IPv4(192, 168, 123, 2)

	client, closer := GetNet(t, &wgapi.ClientConfig{
		Private:   keys.clientPrivate,
		Public:    keys.serverPublic,
		PreShared: keys.shared,
		Endpoint: net.UDPAddr{
			IP:   net.IPv4(127, 0, 0, 1),
			Port: int(wgPort),
		},
		PersistentKeepalive: uint32(wgapi.DefaultPersistentKeepalive),
		AllowedIPs:          []net.IPNet{net.IPNet(wgapi.EmptySubnet)},
	})
	defer closer()

	server, closer := GetNet(t, &wgapi.ServerConfig{
		Private:    keys.serverPrivate,
		ListenPort: wgPort,
		Peers: []*wgapi.ServerPeer{
			{
				Public:    keys.clientPublic,
				PreShared: keys.shared,
				Address:   clientPublic,
			},
		},
	})
	defer closer()

	remoteAddrChan := make(chan net.IP)
	errs := make(chan error)

	rand8 := func() uint8 { return uint8(rand.Intn(math.MaxUint8) + 1) }
	remoteAddr := net.IPv4(rand8(), rand8(), rand8(), 1)
	remotePort := uint16(rand.Intn(math.MaxUint16) + 1)
	go func() {
		ln, err := client.Net(clientPublic).ListenTCP(remotePort)
		if err != nil {
			errs <- err
			return
		}
		defer ln.Close()

		conn, err := ln.Accept()
		if err != nil {
			errs <- err
			return
		}
		defer conn.Close()
		remoteAddrChan <- conn.RemoteAddr().(*net.TCPAddr).IP
	}()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		conn, err := server.Net(remoteAddr).DialTCP(ctx, &net.TCPAddr{IP: clientPublic, Port: int(remotePort)})
		if err != nil {
			errs <- err
			return
		}
		defer conn.Close()
	}()

	tm := time.NewTimer(time.Second * 30)
	defer tm.Stop()
	select {
	case err := <-errs:
		t.Fatal(err)
	case addr := <-remoteAddrChan:
		if !remoteAddr.Equal(addr) {
			t.Fatalf("remote is %s expected %s", addr, remoteAddr)
		}
	case <-tm.C:
		t.Fatal("timeout")
	}
}

type keys struct {
	clientPrivate wgapi.KeyPrivate
	clientPublic  wgapi.KeyPublic
	serverPrivate wgapi.KeyPrivate
	serverPublic  wgapi.KeyPublic
	shared        wgapi.KeyPreShared
}

func generateKeys(t *testing.T) *keys {
	t.Helper()

	var k keys

	clientPrivate, err := wgapi.NewKey()
	if err != nil {
		t.Fatal(err)
	}
	k.clientPrivate = *clientPrivate

	clientPublic, err := clientPrivate.Public()
	if err != nil {
		t.Fatal(err)
	}
	k.clientPublic = *clientPublic

	serverPrivate, err := wgapi.NewKey()
	if err != nil {
		t.Fatal(err)
	}
	k.serverPrivate = *serverPrivate

	serverPublic, err := serverPrivate.Public()
	if err != nil {
		t.Fatal(err)
	}
	k.serverPublic = *serverPublic

	sharedKey, err := wgapi.NewPreShared()
	if err != nil {
		t.Fatal(err)
	}
	k.shared = *sharedKey

	return &k
}

func GetNet(t *testing.T, cfg wgapi.Configurable) (*wg4d.Netstack, func()) {
	t.Helper()
	stack, err := wg4d.NewNetstack(wg4d.DefaultMTU, wg4d.DefaultBatchSize, wg4d.DefaultChannelSize)
	if err != nil {
		t.Fatal(err)
	}

	conn, err := wg4d.New(stack.Device(), wg4d.DefaultBind(), wg4d.NoopLogger(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	return stack, func() {
		err := conn.Close()
		err = errors.Join(err, stack.Close())
		if err != nil {
			t.Fatal(err)
		}
	}
}
