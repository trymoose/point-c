package wgapi

import (
	"io"
	"net"
)

type (
	ServerConfig struct {
		Private    KeyPrivate
		Peers      []*ServerPeer
		ListenPort uint16
	}
	ServerPeer struct {
		Public    KeyPublic
		PreShared KeyPreShared
		Address   net.IP
	}
)

func (cfg *ServerConfig) WGConfig() io.Reader {
	conf := IPC{
		PrivateKey(cfg.Private),
		ListenPort(cfg.ListenPort),
	}.WGConfig()

	for _, peer := range cfg.Peers {
		conf = io.MultiReader(conf, peer.WGConfig())
	}

	return conf
}

func (cfg *ServerPeer) WGConfig() io.Reader {
	return IPC{
		PublicKey(cfg.Public),
		PresharedKey(cfg.PreShared),
		IdentitySubnet(cfg.Address),
	}.WGConfig()
}
