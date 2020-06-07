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
	"bytes"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/xgfone/bt/bencode"
	"github.com/xgfone/bt/metainfo"
)

// Predefine some errors about extension support.
var (
	ErrNotSupportDHT      = fmt.Errorf("not support DHT extension")
	ErrNotSupportFast     = fmt.Errorf("not support Fast extension")
	ErrNotSupportExtended = fmt.Errorf("not support Extended extension")
)

// Bep3Handler is used to handle the BEP 3 type message if Handler has also
// implemented the interface.
type Bep3Handler interface {
	Choke(pc *PeerConn) error
	Unchoke(pc *PeerConn) error
	Interested(pc *PeerConn) error
	NotInterested(pc *PeerConn) error
	Have(pc *PeerConn, index uint32) error
	Bitfield(pc *PeerConn, bits []bool) error
	Request(pc *PeerConn, index uint32, begin uint32, length uint32) error
	Piece(pc *PeerConn, index uint32, begin uint32, piece []byte) error
	Cancel(pc *PeerConn, index uint32, begin uint32, length uint32) error
}

// Bep5Handler is used to handle the BEP 5 type message if Handler has also
// implemented the interface.
//
// Notice: the server must enable the DHT extension bit.
type Bep5Handler interface {
	Port(pc *PeerConn, port uint16) error
}

// Bep6Handler is used to handle the BEP 6 type message if Handler has also
// implemented the interface.
//
// Notice: the server must enable the Fast extension bit.
type Bep6Handler interface {
	HaveAll(pc *PeerConn) error
	HaveNone(pc *PeerConn) error
	Suggest(pc *PeerConn, index uint32) error
	AllowedFast(pc *PeerConn, index uint32) error
	Reject(pc *PeerConn, index uint32, begin uint32, length uint32) error
}

// Bep10Handler is used to handle the BEP 10 extended peer message
// if Handler has also implemented the interface.
//
// Notice: the server must enable the Extended extension bit.
type Bep10Handler interface {
	OnHandShake(conn *PeerConn, exthmsg ExtendedHandshakeMsg) error
	OnPayload(conn *PeerConn, extid uint8, payload []byte) error
}

// PeerConn is used to manage the connection to the peer.
type PeerConn struct {
	net.Conn
	ExtensionBits
	ID metainfo.Hash

	// Timeout is used to control the timeout of reading/writing the message.
	//
	// The default is 0, which represents no timeout.
	Timeout time.Duration

	// MaxLength is used to limit the maximum number of the message body.
	//
	// The default is 0, which represents no limit.
	MaxLength uint32

	// These two states is controlled by the local client peer.
	//
	// Choked is used to indicate whether or not the local client has choked
	// the remote peer, that's, if Choked is true, the local client will
	// discard all the pending requests from the remote peer and not answer
	// any requests until the local client is unchoked.
	//
	// Interested is used to indicate whether or not the local client is
	// interested in something the remote peer has to offer, that's,
	// if Interested is true, the local client will begin requesting blocks
	// when the remote client unchokes them.
	//
	Choked     bool // The default should be true.
	Interested bool

	// These two states is controlled by the remote peer.
	//
	// PeerChoked is used to indicate whether or not the remote peer has choked
	// the local client, that's, if PeerChoked is true, the remote peer will
	// discard all the pending requests from the local client and not answer
	// any requests until the remote peer is unchoked.
	//
	// PeerInterested is used to indicate whether or not the remote peer is
	// interested in something the local client has to offer, that's,
	// if PeerInterested is true, the remote peer will begin requesting blocks
	// when the local client unchokes them.
	//
	PeerChoked     bool // The default should be true.
	PeerInterested bool

	// Data is used to store the context data associated with the connection.
	Data interface{}
}

// NewPeerConn returns a new PeerConn.
func NewPeerConn(id metainfo.Hash, conn net.Conn) *PeerConn {
	return &PeerConn{ID: id, Conn: conn, Choked: true, PeerChoked: true}
}

// NewPeerConnByDial returns a new PeerConn by dialing to addr with the "tcp" network.
func NewPeerConnByDial(id metainfo.Hash, addr string) (pc *PeerConn, err error) {
	conn, err := net.Dial("tcp", addr)
	if err == nil {
		pc = NewPeerConn(id, conn)
	}
	return
}

func (pc *PeerConn) setReadTimeout() {
	if pc.Timeout > 0 {
		pc.Conn.SetReadDeadline(time.Now().Add(pc.Timeout))
	}
}

func (pc *PeerConn) setWriteTimeout() {
	if pc.Timeout > 0 {
		pc.Conn.SetWriteDeadline(time.Now().Add(pc.Timeout))
	}
}

// SetChoked sets the Choked state of the local client peer.
//
// Notice: if the current state is not Choked, it will send a Choked message
// to the remote peer.
func (pc *PeerConn) SetChoked() (err error) {
	if !pc.Choked {
		if err = pc.SendChoke(); err == nil {
			pc.Choked = true
		}
	}
	return
}

// SetUnchoked sets the Unchoked state of the local client peer.
//
// Notice: if the current state is not Unchoked, it will send a Unchoked message
// to the remote peer.
func (pc *PeerConn) SetUnchoked() (err error) {
	if pc.Choked {
		if err = pc.SendUnchoke(); err == nil {
			pc.Choked = false
		}
	}
	return
}

// SetInterested sets the Interested state of the local client peer.
//
// Notice: if the current state is not Interested, it will send a Interested
// message to the remote peer.
func (pc *PeerConn) SetInterested() (err error) {
	if !pc.Interested {
		if err = pc.SendInterested(); err == nil {
			pc.Interested = true
		}
	}
	return
}

// SetNotInterested sets the NotInterested state of the local client peer.
//
// Notice: if the current state is not NotInterested, it will send
// a NotInterested message to the remote peer.
func (pc *PeerConn) SetNotInterested() (err error) {
	if pc.Interested {
		if err = pc.SendNotInterested(); err == nil {
			pc.Interested = false
		}
	}
	return
}

// Handshake does a handshake with the peer.
//
// BEP 3
func (pc *PeerConn) Handshake(infoHash metainfo.Hash) (HandshakeMsg, error) {
	m := HandshakeMsg{ExtensionBits: pc.ExtensionBits, PeerID: pc.ID, InfoHash: infoHash}
	pc.setReadTimeout()
	return Handshake(pc.Conn, m)
}

// ReadMsg reads the message.
//
// BEP 3
func (pc *PeerConn) ReadMsg() (m Message, err error) {
	pc.setReadTimeout()
	err = m.Decode(pc.Conn, pc.MaxLength)
	return
}

// WriteMsg writes the message to the peer.
//
// BEP 3
func (pc *PeerConn) WriteMsg(m Message) (err error) {
	buf := bytes.NewBuffer(make([]byte, 0, 128))
	if err = m.Encode(buf); err == nil {
		pc.setWriteTimeout()

		var n int
		if n, err = pc.Conn.Write(buf.Bytes()); err == nil && n < buf.Len() {
			err = io.ErrShortWrite
		}
	}
	return
}

// SendKeepalive sends a Keepalive message to the peer.
//
// BEP 3
func (pc *PeerConn) SendKeepalive() error {
	return pc.WriteMsg(Message{Keepalive: true})
}

// SendChoke sends a Choke message to the peer.
//
// BEP 3
func (pc *PeerConn) SendChoke() error {
	return pc.WriteMsg(Message{Type: Choke})
}

// SendUnchoke sends a Unchoke message to the peer.
//
// BEP 3
func (pc *PeerConn) SendUnchoke() error {
	return pc.WriteMsg(Message{Type: Unchoke})
}

// SendInterested sends a Interested message to the peer.
//
// BEP 3
func (pc *PeerConn) SendInterested() error {
	return pc.WriteMsg(Message{Type: Interested})
}

// SendNotInterested sends a NotInterested message to the peer.
//
// BEP 3
func (pc *PeerConn) SendNotInterested() error {
	return pc.WriteMsg(Message{Type: NotInterested})
}

// SendBitfield sends a Bitfield message to the peer.
//
// BEP 3
func (pc *PeerConn) SendBitfield(bits []bool) error {
	return pc.WriteMsg(Message{Type: Bitfield, Bitfield: bits})
}

// SendHave sends a Have message to the peer.
//
// BEP 3
func (pc *PeerConn) SendHave(index uint32) error {
	return pc.WriteMsg(Message{Type: Have, Index: index})
}

// SendRequest sends a Request message to the peer.
//
// BEP 3
func (pc *PeerConn) SendRequest(index, begin, length uint32) error {
	return pc.WriteMsg(Message{Type: Request, Index: index, Begin: begin, Length: length})
}

// SendCancel sends a Cancel message to the peer.
//
// BEP 3
func (pc *PeerConn) SendCancel(index, begin, length uint32) error {
	return pc.WriteMsg(Message{Type: Cancel, Index: index, Begin: begin, Length: length})
}

// SendPiece sends a Piece message to the peer.
//
// BEP 3
func (pc *PeerConn) SendPiece(index, begin uint32, piece []byte) error {
	return pc.WriteMsg(Message{Type: Piece, Index: index, Begin: begin, Piece: piece})
}

// SendPort sends a Port message to the peer.
//
// BEP 5
func (pc *PeerConn) SendPort(port uint16) error {
	return pc.WriteMsg(Message{Type: Port, Port: port})
}

// SendHaveAll sends a HaveAll message to the peer.
//
// BEP 6
func (pc *PeerConn) SendHaveAll() error {
	return pc.WriteMsg(Message{Type: HaveAll})
}

// SendHaveNone sends a HaveNone message to the peer.
//
// BEP 6
func (pc *PeerConn) SendHaveNone() error {
	return pc.WriteMsg(Message{Type: HaveNone})
}

// SendSuggest sends a Suggest message to the peer.
//
// BEP 6
func (pc *PeerConn) SendSuggest(index uint32) error {
	return pc.WriteMsg(Message{Type: Suggest, Index: index})
}

// SendReject sends a Reject message to the peer.
//
// BEP 6
func (pc *PeerConn) SendReject(index, begin, length uint32) error {
	return pc.WriteMsg(Message{Type: Reject, Index: index, Begin: begin, Length: length})
}

// SendAllowedFast sends a AllowedFast message to the peer.
//
// BEP 6
func (pc *PeerConn) SendAllowedFast(index uint32) error {
	return pc.WriteMsg(Message{Type: AllowedFast, Index: index})
}

// SendExtHandshakeMsg sends the Extended Handshake message to the peer.
//
// BEP 10
func (pc *PeerConn) SendExtHandshakeMsg(m ExtendedHandshakeMsg) (err error) {
	buf := bytes.NewBuffer(make([]byte, 0, 128))
	if err = bencode.NewEncoder(buf).Encode(m); err == nil {
		err = pc.SendExtMsg(ExtendedIDHandshake, buf.Bytes())
	}
	return
}

// SendExtMsg sends the Extended message with the extended id and the payload.
//
// BEP 10
func (pc *PeerConn) SendExtMsg(extID uint8, payload []byte) error {
	return pc.WriteMsg(Message{Type: Extended, ExtendedID: extID, ExtendedPayload: payload})
}

// HandleMessage calls the method of the handler to handle the message.
//
// If handler has also implemented the interfaces Bep3Handler, Bep5Handler,
// Bep6Handler or Bep10Handler, their methods will be called instead of
// Handler.OnMessage for the corresponding type message.
func (pc *PeerConn) HandleMessage(msg Message, handler Handler) (err error) {
	if msg.Keepalive {
		return
	}

	switch msg.Type {
	// BEP 3 - The BitTorrent Protocol Specification
	case Choke:
		pc.PeerChoked = true
		if h, ok := handler.(Bep3Handler); ok {
			err = h.Choke(pc)
		} else {
			err = handler.OnMessage(pc, msg)
		}
	case Unchoke:
		pc.PeerChoked = false
		if h, ok := handler.(Bep3Handler); ok {
			err = h.Unchoke(pc)
		} else {
			err = handler.OnMessage(pc, msg)
		}
	case Interested:
		pc.PeerInterested = true
		if h, ok := handler.(Bep3Handler); ok {
			err = h.Interested(pc)
		} else {
			err = handler.OnMessage(pc, msg)
		}
	case NotInterested:
		pc.PeerInterested = false
		if h, ok := handler.(Bep3Handler); ok {
			err = h.NotInterested(pc)
		} else {
			err = handler.OnMessage(pc, msg)
		}
	case Have:
		if h, ok := handler.(Bep3Handler); ok {
			err = h.Have(pc, msg.Index)
		} else {
			err = handler.OnMessage(pc, msg)
		}
	case Bitfield:
		if h, ok := handler.(Bep3Handler); ok {
			err = h.Bitfield(pc, msg.Bitfield)
		} else {
			err = handler.OnMessage(pc, msg)
		}
	case Request:
		if h, ok := handler.(Bep3Handler); ok {
			err = h.Request(pc, msg.Index, msg.Begin, msg.Length)
		} else {
			err = handler.OnMessage(pc, msg)
		}
	case Piece:
		if h, ok := handler.(Bep3Handler); ok {
			err = h.Piece(pc, msg.Index, msg.Begin, msg.Piece)
		} else {
			err = handler.OnMessage(pc, msg)
		}
	case Cancel:
		if h, ok := handler.(Bep3Handler); ok {
			err = h.Cancel(pc, msg.Index, msg.Begin, msg.Length)
		} else {
			err = handler.OnMessage(pc, msg)
		}

	// BEP 5 - DHT Protocol
	case Port:
		if !pc.IsSupportDHT() {
			return ErrNotSupportDHT
		} else if h, ok := handler.(Bep5Handler); ok {
			err = h.Port(pc, msg.Port)
		} else {
			err = handler.OnMessage(pc, msg)
		}

	// BEP 6 - Fast Extension
	case Suggest:
		if !pc.IsSupportFast() {
			return ErrNotSupportFast
		} else if h, ok := handler.(Bep6Handler); ok {
			err = h.Suggest(pc, msg.Index)
		} else {
			err = handler.OnMessage(pc, msg)
		}
	case HaveAll:
		if !pc.IsSupportFast() {
			return ErrNotSupportFast
		} else if h, ok := handler.(Bep6Handler); ok {
			err = h.HaveAll(pc)
		} else {
			err = handler.OnMessage(pc, msg)
		}
	case HaveNone:
		if !pc.IsSupportFast() {
			return ErrNotSupportFast
		} else if h, ok := handler.(Bep6Handler); ok {
			err = h.HaveNone(pc)
		} else {
			err = handler.OnMessage(pc, msg)
		}
	case Reject:
		if !pc.IsSupportFast() {
			return ErrNotSupportFast
		} else if h, ok := handler.(Bep6Handler); ok {
			err = h.Reject(pc, msg.Index, msg.Begin, msg.Length)
		} else {
			err = handler.OnMessage(pc, msg)
		}
	case AllowedFast:
		if !pc.IsSupportFast() {
			return ErrNotSupportFast
		} else if h, ok := handler.(Bep6Handler); ok {
			err = h.AllowedFast(pc, msg.Index)
		} else {
			err = handler.OnMessage(pc, msg)
		}

	// BEP 10 - Extension Protocol
	case Extended:
		if !pc.IsSupportExtended() {
			return ErrNotSupportExtended
		} else if h, ok := handler.(Bep10Handler); ok {
			err = pc.handleExtMsg(h, msg)
		} else {
			err = handler.OnMessage(pc, msg)
		}

	// Other
	default:
		err = handler.OnMessage(pc, msg)
	}

	return
}

func (pc *PeerConn) handleExtMsg(h Bep10Handler, m Message) (err error) {
	if m.ExtendedID == ExtendedIDHandshake {
		var ehmsg ExtendedHandshakeMsg
		if err = bencode.DecodeBytes(m.ExtendedPayload, &ehmsg); err == nil {
			err = h.OnHandShake(pc, ehmsg)
		}
	} else {
		err = h.OnPayload(pc, m.ExtendedID, m.ExtendedPayload)
	}

	return
}
