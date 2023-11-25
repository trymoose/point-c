package channel_listener

import (
	"errors"
	"net"
	"testing"
	"time"
)

type testConn struct{}

func (*testConn) Read([]byte) (int, error)         { return 0, nil }
func (*testConn) Write([]byte) (int, error)        { return 0, nil }
func (*testConn) Close() error                     { return nil }
func (*testConn) LocalAddr() net.Addr              { return nil }
func (*testConn) RemoteAddr() net.Addr             { return nil }
func (*testConn) SetDeadline(time.Time) error      { return nil }
func (*testConn) SetReadDeadline(time.Time) error  { return nil }
func (*testConn) SetWriteDeadline(time.Time) error { return nil }

func TestListener(t *testing.T) {
	t.Run("new & close", func(t *testing.T) {
		conns := make(chan net.Conn)
		conn := new(testConn)
		go func() { conns <- conn }()
		ln := New(conns, nil)
		done := make(chan struct{})
		go func() {
			defer close(done)
			c, err := ln.Accept()
			if c != nil {
				t.Fail()
			}
			if !errors.Is(err, net.ErrClosed) {
				t.Fail()
			}
		}()
		if ln.Close() != nil {
			t.Fail()
		}
		select {
		case <-time.After(time.Second * 10):
			t.Fail()
		case <-done:
		}
	})

	t.Run("addr", func(t *testing.T) {
		addr := &net.TCPAddr{}
		ln := New(make(<-chan net.Conn), addr)
		if addr != ln.Addr() {
			t.Fail()
		}
	})

	t.Run("accept", func(t *testing.T) {
		conns := make(chan net.Conn)
		conn := new(testConn)
		ln := New(conns, nil)

		done := make(chan struct{})
		go func() {
			defer close(done)
			c, err := ln.Accept()
			if err != nil {
				t.Fail()
			}
			if c == nil {
				t.Fail()
			} else if c != conn {
				t.Fail()
			}

			c, err = ln.Accept()
			if err == nil {
				t.Fail()
			}
			if c != nil {
				t.Fail()
			}
		}()

		conns <- conn
		if ln.Close() != nil {
			t.Fail()
		}

		select {
		case <-time.After(time.Second * 10):
			t.Fail()
		case <-ln.Done():
		case <-done:
		}
	})

	t.Run("closed input chan", func(t *testing.T) {
		conns := make(chan net.Conn)
		close(conns)
		ln := New(conns, nil)
		c, err := ln.Accept()
		if c != nil {
			t.Fail()
		}
		if !errors.Is(err, net.ErrClosed) {
			t.Fail()
		}
	})
}
