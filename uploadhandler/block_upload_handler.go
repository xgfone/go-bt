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

package blockdownload

import (
	"github.com/xgfone/bt/metainfo"
	pp "github.com/xgfone/bt/peerprotocol"
)

// BlockUploadHandler is used to downloads the files in the torrent file.
type BlockUploadHandler struct {
	pp.NoopHandler
	pp.NoopBep3Handler
	pp.NoopBep6Handler

	Info         metainfo.Info                              // Required
	OnBlock      func(index, offset uint32, b []byte) error // Required
	RespondBlock func(c *pp.PeerConn) error                 // Required
}

// NewBlockUploadHandler returns a new BlockUploadHandler.
func NewBlockUploadHandler(info metainfo.Info, onBlock func(pieceIndex, pieceOffset uint32, b []byte) error, respondBlock func(c *pp.PeerConn) error) BlockUploadHandler {
	return BlockUploadHandler{
		Info:         info,
		OnBlock:      onBlock,
		RespondBlock: respondBlock,
	}
}

// OnHandShake implements the interface Handler#OnHandShake.
//
// Notice: it uses the field Data to store the inner data, you mustn't override
// it.
func (fd BlockUploadHandler) OnHandShake(c *pp.PeerConn) (err error) {
	if err = c.SetUnchoked(); err == nil {
		err = c.SetInterested()
	}
	return
}

/// ---------------------------------------------------------------------------
/// BEP 3

func (fd BlockUploadHandler) respond(pc *pp.PeerConn) (err error) {
	if pc.PeerChoked {
		err = pp.ErrChoked
	} else {
		err = fd.RespondBlock(pc)
	}
	return
}

// Piece implements the interface Bep3Handler#Piece.
func (fd BlockUploadHandler) Piece(c *pp.PeerConn, i, b uint32, p []byte) (err error) {
	if err = fd.OnBlock(i, b, p); err == nil {
		err = fd.respond(c)
	}
	return
}

// Unchoke implements the interface Bep3Handler#Unchoke.
func (fd BlockUploadHandler) Unchoke(pc *pp.PeerConn) (err error) {
	return fd.respond(pc)
}

// Have implements the interface Bep3Handler#Have.
func (fd BlockUploadHandler) Have(pc *pp.PeerConn, index uint32) (err error) {
	pc.BitField.Set(index)
	return
}

/// ---------------------------------------------------------------------------
/// BEP 6

// HaveAll implements the interface Bep6Handler#HaveAll.
func (fd BlockUploadHandler) HaveAll(pc *pp.PeerConn) (err error) {
	pc.BitField = pp.NewBitField(fd.Info.CountPieces(), true)
	return
}

// Reject implements the interface Bep6Handler#Reject.
func (fd BlockUploadHandler) Reject(pc *pp.PeerConn, index, begin, length uint32) (err error) {
	pc.BitField.Unset(index)
	return fd.respond(pc)
}
