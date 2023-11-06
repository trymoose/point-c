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

type Connection struct {
	dev   *device.Device
	close sync.Once
}

func SlogLogger(logger *slog.Logger) *device.Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return &device.Logger{
		Verbosef: func(format string, args ...any) { logger.Debug(fmt.Sprintf(format, args...)) },
		Errorf:   func(format string, args ...any) { logger.Error(fmt.Sprintf(format, args...)) },
	}
}

func NoopLogger() *device.Logger {
	return &device.Logger{
		Verbosef: func(string, ...any) {},
		Errorf:   func(string, ...any) {},
	}
}

func DefaultBind() conn.Bind {
	return conn.NewDefaultBind()
}

func New(tun tun.Device, bind conn.Bind, logger *device.Logger, cfg wgapi.Configurable) (*Connection, error) {
	c := &Connection{dev: device.NewDevice(tun, bind, logger)}

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

func (c *Connection) GetConfig() (wgapi.IPC, error) {
	var ipc wgapi.IPCGet
	if err := c.dev.IpcGetOperation(&ipc); err != nil {
		return nil, err
	}
	return ipc.Value()
}

func (c *Connection) Close() (err error) {
	c.close.Do(func() {
		err = c.dev.Down()
		c.dev.Close()
	})
	return
}
