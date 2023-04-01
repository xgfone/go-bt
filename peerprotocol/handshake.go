// Copyright 2020 xgfone
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package peerprotocol

import (
	"encoding/hex"
	"fmt"
	"io"

	"github.com/xgfone/go-bt/metainfo"
)

var errInvalidProtocolHeader = fmt.Errorf("unexpected peer protocol header string")

// Predefine some known extension bits.
const (
	ExtensionBitDHT      = 0  // BEP 5
	ExtensionBitFast     = 2  // BEP 6
	ExtensionBitExtended = 20 // BEP 10

	ExtensionBitMax = 64
)

// ExtensionBits is the reserved bytes to be used by all extensions.
//
// BEP 10: The bit is counted starting at 0 from right to left.
type ExtensionBits [8]byte

// String returns the hex string format.
func (eb ExtensionBits) String() string {
	return hex.EncodeToString(eb[:])
}

// Set sets the bit to 1, that's, to set it to be on.
func (eb *ExtensionBits) Set(bit uint) {
	eb[7-bit/8] |= 1 << (bit % 8)
}

// Unset sets the bit to 0, that's, to set it to be off.
func (eb *ExtensionBits) Unset(bit uint) {
	eb[7-bit/8] &^= 1 << (bit % 8)
}

// IsSet reports whether the bit is on.
func (eb ExtensionBits) IsSet(bit uint) (yes bool) {
	return eb[7-bit/8]&(1<<(bit%8)) != 0
}

// IsSupportDHT reports whether ExtensionBitDHT is set.
func (eb ExtensionBits) IsSupportDHT() (yes bool) {
	return eb.IsSet(ExtensionBitDHT)
}

// IsSupportFast reports whether ExtensionBitFast is set.
func (eb ExtensionBits) IsSupportFast() (yes bool) {
	return eb.IsSet(ExtensionBitFast)
}

// IsSupportExtended reports whether ExtensionBitExtended is set.
func (eb ExtensionBits) IsSupportExtended() (yes bool) {
	return eb.IsSet(ExtensionBitExtended)
}

// HandshakeMsg is the message used by the handshake
type HandshakeMsg struct {
	ExtensionBits

	PeerID   metainfo.Hash
	InfoHash metainfo.Hash
}

// NewHandshakeMsg returns a new HandshakeMsg.
func NewHandshakeMsg(peerID, infoHash metainfo.Hash, es ...ExtensionBits) HandshakeMsg {
	var e ExtensionBits
	if len(es) > 0 {
		e = es[0]
	}
	return HandshakeMsg{ExtensionBits: e, PeerID: peerID, InfoHash: infoHash}
}

// Handshake finishes the handshake with the peer.
//
// InfoHash may be ZERO, and it will read it from the peer then send it back
// to the peer.
//
// BEP 3
func Handshake(sock io.ReadWriter, msg HandshakeMsg) (ret HandshakeMsg, err error) {
	var read bool
	if msg.InfoHash.IsZero() {
		if err = getPeerHandshakeMsg(sock, &ret); err != nil {
			return
		}
		read = true
		msg.InfoHash = ret.InfoHash
	}

	if _, err = io.WriteString(sock, ProtocolHeader); err != nil {
		return
	}
	if _, err = sock.Write(msg.ExtensionBits[:]); err != nil {
		return
	}
	if _, err = sock.Write(msg.InfoHash[:]); err != nil {
		return
	}
	if _, err = sock.Write(msg.PeerID[:]); err != nil {
		return
	}

	if !read {
		err = getPeerHandshakeMsg(sock, &ret)
	}
	return
}

func getPeerHandshakeMsg(sock io.Reader, ret *HandshakeMsg) (err error) {
	var b [68]byte
	if _, err = io.ReadFull(sock, b[:]); err != nil {
		return
	} else if string(b[:20]) != ProtocolHeader {
		return errInvalidProtocolHeader
	}

	copy(ret.ExtensionBits[:], b[20:28])
	copy(ret.InfoHash[:], b[28:48])
	copy(ret.PeerID[:], b[48:68])
	return
}
