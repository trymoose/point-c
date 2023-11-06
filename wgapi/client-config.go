package wgapi

import (
	"io"
	"net"
)

type ClientConfig struct {
	Private             KeyPrivate // Client private key
	Public              KeyPublic  // Server public key
	PreShared           KeyPreShared
	Endpoint            net.UDPAddr
	PersistentKeepalive uint32
	AllowedIPs          []net.IPNet
}

func (cfg *ClientConfig) WGConfig() io.Reader {
	conf := IPC{
		PrivateKey(cfg.Private),
		ReplacePeers{},
		PublicKey(cfg.Public),
		Endpoint(cfg.Endpoint),
		PresharedKey(cfg.PreShared),
		PersistentKeepalive(cfg.PersistentKeepalive),
	}.WGConfig()

	for _, allowed := range cfg.AllowedIPs {
		conf = io.MultiReader(conf, IPC{
			AllowedIP(allowed),
		}.WGConfig())
	}
	return conf
}
