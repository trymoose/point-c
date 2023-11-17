package wglog

import (
	"golang.zx2c4.com/wireguard/device"
)

type Logger = device.Logger

// Noop is a logger that does not output anything.
func Noop() *Logger {
	return &Logger{
		Verbosef: func(string, ...any) {},
		Errorf:   func(string, ...any) {},
	}
}
