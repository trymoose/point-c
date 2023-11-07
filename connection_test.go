package wg4d_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/trymoose/wg4d"
	"github.com/trymoose/wg4d/device"
	"github.com/trymoose/wg4d/wgapi"
	"github.com/trymoose/wg4d/wgapi/wgconfig"
	"golang.org/x/exp/rand"
	"math"
	"net"
	"testing"
	"time"
)

const logWG = false

func TestTCPConnection(t *testing.T) {
	keys := generateKeys(t)
	wgPort := uint16(51820)
	clientPublic := net.IPv4(192, 168, 123, 2)

	clientConfig := &wgconfig.Client{
		Private:   keys.clientPrivate,
		Public:    keys.serverPublic,
		PreShared: keys.shared,
		Endpoint:  net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: int(wgPort)},
	}
	clientConfig.AllowAllIPs()
	clientConfig.DefaultPersistentKeepAlive()

	client, closer := GetNet(t, clientConfig)
	defer closer()

	serverConfig := &wgconfig.Server{
		Private: keys.serverPrivate,
	}
	serverConfig.DefaultListenPort()
	serverConfig.AddPeer(keys.clientPublic, keys.shared, clientPublic)

	server, closer := GetNet(t, serverConfig)
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

	tm := time.NewTimer(time.Second * 35)
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
	clientPrivate wgapi.PrivateKey
	clientPublic  wgapi.PublicKey
	serverPrivate wgapi.PrivateKey
	serverPublic  wgapi.PublicKey
	shared        wgapi.PresharedKey
}

func generateKeys(t *testing.T) *keys {
	t.Helper()

	var k keys

	clientPrivate, clientPublic, err := wgapi.NewPrivatePublic()
	if err != nil {
		t.Fatal(err)
	}
	k.clientPrivate, k.clientPublic = clientPrivate, clientPublic

	serverPrivate, serverPublic, err := wgapi.NewPrivatePublic()
	if err != nil {
		t.Fatal(err)
	}
	k.serverPrivate, k.serverPublic = serverPrivate, serverPublic

	sharedKey, err := wgapi.NewPreshared()
	if err != nil {
		t.Fatal(err)
	}
	k.shared = sharedKey

	return &k
}

func GetNet(t *testing.T, cfg wgapi.Configurable) (*device.Device, func()) {
	t.Helper()
	stack, err := device.NewDefault()
	if err != nil {
		t.Fatal(err)
	}

	logger := wg4d.NoopLogger()
	if logWG {
		logger.Errorf = func(format string, args ...any) {
			t.Helper()
			t.Logf("ERROR: %s", fmt.Sprintf(format, args...))
		}
		logger.Verbosef = func(format string, args ...any) {
			t.Helper()
			t.Logf("INFO:  %s", fmt.Sprintf(format, args...))
		}
	}

	conn, err := wg4d.New(stack.Device(), wg4d.DefaultBind(), logger, cfg)
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
