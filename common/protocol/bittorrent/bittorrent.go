package bittorrent

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/xtls/xray-core/common"
)

type SniffHeader struct{}

func (h *SniffHeader) Protocol() string {
	return "bittorrent"
}

func (h *SniffHeader) Domain() string {
	return ""
}

var errNotBittorrent = errors.New("not bittorrent header")

func SniffBittorrent(b []byte) (*SniffHeader, error) {
	if len(b) < 20 {
		return nil, common.ErrNoClue
	}

	if b[0] == 19 && string(b[1:20]) == "BitTorrent protocol" {
		return &SniffHeader{}, nil
	}

	return nil, errNotBittorrent
}

// SniffBittorrentUDP recognises the UDP side of BitTorrent from the first
// datagram of a flow: UDP tracker handshakes (BEP 15), DHT messages (BEP 5)
// and uTP connection requests (BEP 29). A datagram is always complete, so a
// miss is a definite answer rather than ErrNoClue: waiting for more data
// would only delay the flow.
func SniffBittorrentUDP(b []byte) (*SniffHeader, error) {
	if isUDPTrackerConnect(b) || isDHTMessage(b) || isUTPSyn(b) {
		return &SniffHeader{}, nil
	}
	return nil, errNotBittorrent
}

// udpTrackerMagic is the protocol id opening every UDP tracker connect
// request.
const udpTrackerMagic = 0x41727101980

func isUDPTrackerConnect(b []byte) bool {
	return len(b) >= 16 &&
		binary.BigEndian.Uint64(b[0:8]) == udpTrackerMagic &&
		binary.BigEndian.Uint32(b[8:12]) == 0 // action: connect
}

// isDHTMessage matches a KRPC message: a bencoded dictionary carrying the
// sender's 20-byte node id and the message type key.
func isDHTMessage(b []byte) bool {
	return len(b) >= 20 && b[0] == 'd' && b[len(b)-1] == 'e' &&
		bytes.Contains(b, []byte("2:id20:")) && bytes.Contains(b, []byte("1:y1:"))
}

const (
	// utpHeaderSize: type/version, extension, connection id, two
	// timestamps, window size, seq and ack numbers.
	utpHeaderSize = 20
	// utpSynV1 is ST_SYN (type 4) of protocol version 1.
	utpSynV1 = 0x41
)

// isUTPSyn matches the connection request that opens every outgoing uTP
// flow. A SYN carries no payload, so the header and its extension chain must
// cover the whole datagram exactly, which keeps game or voice packets that
// happen to start with the same byte from being taken for it.
func isUTPSyn(b []byte) bool {
	if len(b) < utpHeaderSize || b[0] != utpSynV1 {
		return false
	}
	ext := b[1]
	rest := b[utpHeaderSize:]
	for ext != 0 {
		if len(rest) < 2 {
			return false
		}
		ext = rest[0]
		size := int(rest[1])
		rest = rest[2:]
		if len(rest) < size {
			return false
		}
		rest = rest[size:]
	}
	return len(rest) == 0
}
