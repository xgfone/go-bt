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

var _ Handler = NoopHandler{}
var _ Bep3Handler = NoopBep3Handler{}
var _ Bep5Handler = NoopBep5Handler{}
var _ Bep6Handler = NoopBep6Handler{}
var _ Bep10Handler = NoopBep10Handler{}

// NoopHandler implements the interface Handler to do nothing,
// which is used to be embedded into other structure to not implement
// the noop interface methods.
type NoopHandler struct{}

// OnClose implements the interface Bep3Handler#OnClose.
func (NoopHandler) OnClose(*PeerConn) {}

// OnHandShake implements the interface Bep3Handler#OnHandShake.
func (NoopHandler) OnHandShake(*PeerConn) error { return nil }

// OnMessage implements the interface Bep3Handler#OnMessage.
func (NoopHandler) OnMessage(*PeerConn, Message) error { return nil }

// NoopBep3Handler implements the interface Bep3Handler to do nothing,
// which is used to be embedded into other structure to not implement
// the noop interface methods.
type NoopBep3Handler struct{}

// Unchoke implements the interface Bep3Handler#Unchoke.
func (NoopBep3Handler) Unchoke(*PeerConn) error { return nil }

// Request implements the interface Bep3Handler#Request.
func (NoopBep3Handler) Request(*PeerConn, uint32, uint32, uint32) error { return nil }

// Have implements the interface Bep3Handler#Have.
func (NoopBep3Handler) Have(*PeerConn, uint32) error { return nil }

// BitField implements the interface Bep3Handler#Bitfield.
func (NoopBep3Handler) BitField(*PeerConn, BitField) error { return nil }

// Piece implements the interface Bep3Handler#Piece.
func (NoopBep3Handler) Piece(*PeerConn, uint32, uint32, []byte) error { return nil }

// Choke implements the interface Bep3Handler#Choke.
func (NoopBep3Handler) Choke(pc *PeerConn) error { return nil }

// Interested implements the interface Bep3Handler#Interested.
func (NoopBep3Handler) Interested(pc *PeerConn) error { return nil }

// NotInterested implements the interface Bep3Handler#NotInterested.
func (NoopBep3Handler) NotInterested(pc *PeerConn) error { return nil }

// Cancel implements the interface Bep3Handler#Cancel.
func (NoopBep3Handler) Cancel(*PeerConn, uint32, uint32, uint32) error { return nil }

// NoopBep5Handler implements the interface Bep5Handler to do nothing,
// which is used to be embedded into other structure to not implement
// the noop interface methods.
type NoopBep5Handler struct{}

// Port implements the interface Bep5Handler#Port.
func (NoopBep5Handler) Port(*PeerConn, uint16) error { return nil }

// NoopBep6Handler implements the interface Bep6Handler to do nothing,
// which is used to be embedded into other structure to not implement
// the noop interface methods.
type NoopBep6Handler struct{}

// HaveAll implements the interface Bep6Handler#HaveAll.
func (NoopBep6Handler) HaveAll(*PeerConn) error { return nil }

// HaveNone implements the interface Bep6Handler#HaveNone.
func (NoopBep6Handler) HaveNone(*PeerConn) error { return nil }

// Suggest implements the interface Bep6Handler#Suggest.
func (NoopBep6Handler) Suggest(*PeerConn, uint32) error { return nil }

// AllowedFast implements the interface Bep6Handler#AllowedFast.
func (NoopBep6Handler) AllowedFast(*PeerConn, uint32) error { return nil }

// Reject implements the interface Bep6Handler#Reject.
func (NoopBep6Handler) Reject(*PeerConn, uint32, uint32, uint32) error { return nil }

// NoopBep10Handler implements the interface Bep10Handler to do nothing,
// which is used to be embedded into other structure to not implement
// the noop interface methods.
type NoopBep10Handler struct{}

// OnHandShake implements the interface Bep10Handler#OnHandShake.
func (NoopBep10Handler) OnHandShake(*PeerConn, ExtendedHandshakeMsg) error { return nil }

// OnPayload implements the interface Bep10Handler#OnPayload.
func (NoopBep10Handler) OnPayload(*PeerConn, uint8, []byte) error { return nil }
