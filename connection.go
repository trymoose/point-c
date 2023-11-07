// Package wg4d helps with the creation and usage of userland wireguard networks.
package wg4d

import (
	"fmt"
	"github.com/trymoose/wg4d/wgapi"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"log/slog"
	"sync"
)

// Wireguard handles configuring and closing a wireguard client/server.
type Wireguard struct {
	dev   *device.Device
	close sync.Once
}

// SlogLogger allows the created of a [device.Logger] backed by a given [slog.Logger].
func SlogLogger(logger *slog.Logger) *device.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return &device.Logger{
		Verbosef: func(format string, args ...any) { logger.Debug(fmt.Sprintf(format, args...)) },
		Errorf:   func(format string, args ...any) { logger.Error(fmt.Sprintf(format, args...)) },
	}
}

// NoopLogger is a logger that does not output anything.
func NoopLogger() *device.Logger {
	return &device.Logger{
		Verbosef: func(string, ...any) {},
		Errorf:   func(string, ...any) {},
	}
}

// DefaultBind is the default wireguard UDP listener..
func DefaultBind() conn.Bind {
	return conn.NewDefaultBind()
}

// New allows the creating of a new wireguard server/client.
func New(tun tun.Device, bind conn.Bind, logger *device.Logger, cfg wgapi.Configurable) (*Wireguard, error) {
	c := &Wireguard{dev: device.NewDevice(tun, bind, logger)}

	if err := c.dev.IpcSetOperation(cfg.WGConfig()); err != nil {
		c.dev.Close()
		return nil, err
	}

	if err := c.dev.Up(); err != nil {
		c.dev.Close()
		return nil, err
	}
	return c, nil
}

// GetConfig gets the raw config from an IPC get=1 operation.
func (c *Wireguard) GetConfig() (wgapi.IPC, error) {
	var ipc wgapi.IPCGet
	if err := c.dev.IpcGetOperation(&ipc); err != nil {
		return nil, err
	}
	return ipc.Value()
}

// Close closes the wireguard server/client, rendering it unusable in the future.
func (c *Wireguard) Close() (err error) {
	c.close.Do(func() {
		err = c.dev.Down()
		c.dev.Close()
	})
	return
}
