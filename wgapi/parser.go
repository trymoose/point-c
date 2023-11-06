package wgapi

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/trymoose/wg4d/wgapi/internal/key"
	"github.com/trymoose/wg4d/wgapi/internal/parser"
	"github.com/trymoose/wg4d/wgapi/internal/value/wgkey"
)

type IPCGet bytes.Buffer

func (get *IPCGet) Write(b []byte) (int, error) { return get.bbuf().Write(b) }
func (get *IPCGet) Reset()                      { get.bbuf().Reset() }
func (get *IPCGet) bbuf() *bytes.Buffer         { return (*bytes.Buffer)(get) }

var kvCutChar = []byte{'='}

func (get *IPCGet) Value() (IPC, error) {
	sc := bufio.NewScanner(get.bbuf())
	sc.Split(parser.ScanLines)
	var ipc IPC
	for i := 0; sc.Scan(); i++ {
		line := sc.Bytes()
		if len(line) == 0 {
			return ipc, nil
		}

		k, v, ok := bytes.Cut(line, kvCutChar)
		if !ok {
			return nil, fmt.Errorf("line %d malformed, expected key=value, got %q", i, line)
		}

		p, ok := parsers[string(k)]
		if !ok {
			return nil, fmt.Errorf("line %d malformed, key %q is not valid", i, k)
		}

		kv, err := p(v)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %q on line %d, %w", k, i, err)
		}
		ipc = append(ipc, kv)
	}
	return ipc, sc.Err()
}

var parsers = map[string]func([]byte) (IPCKeyValue, error){
	key.Endpoint{}.String():              parser.ParseUDPAddr[key.Endpoint],
	key.AllowedIP{}.String():             parser.ParseIPNet[key.AllowedIP],
	key.ReplacePeers{}.String():          parser.ParseTrue[key.ReplacePeers],
	key.Remove{}.String():                parser.ParseTrue[key.Remove],
	key.UpdateOnly{}.String():            parser.ParseTrue[key.UpdateOnly],
	key.ReplaceAllowedIPs{}.String():     parser.ParseTrue[key.ReplaceAllowedIPs],
	key.Get{}.String():                   parser.ParseOne[key.Get],
	key.Set{}.String():                   parser.ParseOne[key.Set],
	key.ProtocolVersion{}.String():       parser.ParseOne[key.ProtocolVersion],
	key.RXBytes{}.String():               parser.ParseUint64[key.RXBytes],
	key.TXBytes{}.String():               parser.ParseUint64[key.TXBytes],
	key.FWMark{}.String():                parser.ParseUint32[key.FWMark],
	key.ListenPort{}.String():            parser.ParseUint16[key.ListenPort],
	key.PersistentKeepalive{}.String():   parser.ParseUint16[key.PersistentKeepalive],
	key.Errno{}.String():                 parser.ParseInt64[key.Errno],
	key.LastHandshakeTimeSec{}.String():  parser.ParseInt64[key.LastHandshakeTimeSec],
	key.LastHandshakeTimeNSec{}.String(): parser.ParseInt64[key.LastHandshakeTimeNSec],
	key.PrivateKey{}.String():            parser.ParseKey[key.PrivateKey, wgkey.Private],
	key.PublicKey{}.String():             parser.ParseKey[key.PublicKey, wgkey.Public],
	key.PresharedKey{}.String():          parser.ParseKey[key.PresharedKey, wgkey.PreShared],
}
