package test_helpers

import (
	"io"
	"net"
	"time"
)

type nopConn struct{}

var nopConnInstance net.Conn = new(nopConn)

func NopConn() net.Conn { return nopConnInstance }

func (n2 *nopConn) Read([]byte) (n int, err error)    { return 0, io.EOF }
func (n2 *nopConn) Write(b []byte) (n int, err error) { return len(b), nil }
func (n2 *nopConn) Close() error                      { return nil }
func (n2 *nopConn) LocalAddr() net.Addr               { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1} }
func (n2 *nopConn) RemoteAddr() net.Addr              { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 2} }
func (n2 *nopConn) SetDeadline(time.Time) error       { return nil }
func (n2 *nopConn) SetReadDeadline(time.Time) error   { return nil }
func (n2 *nopConn) SetWriteDeadline(time.Time) error  { return nil }
