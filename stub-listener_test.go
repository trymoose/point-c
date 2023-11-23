package point_c_test

import (
	pointc "github.com/trymoose/point-c"
	"io"
	"net"
	"testing"
)

func TestStubListener(t *testing.T) {
	ln, err := pointc.StubListener(nil, "", "test", net.ListenConfig{})
	if err != nil {
		t.Fail()
		return
	}
	defer ln.(io.Closer).Close()
}

func TestStubAddr(t *testing.T) {
	ln, err := pointc.StubListener(nil, "", "test", net.ListenConfig{})
	if err != nil {
		t.Fail()
		return
	}
	defer ln.(io.Closer).Close()
	addr := ln.(net.Listener).Addr()
	if addr.Network() != "stub" {
		t.Fail()
	}
	if addr.String() != "test" {
		t.Fail()
	}
}
