// Copyright 2020 xgfone, 2023 idk
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
	"github.com/eyedeekay/go-i2p-bt/metainfo"
	pp "github.com/eyedeekay/go-i2p-bt/peerprotocol"
)

// BlockDownloadHandler is used to downloads the files in the torrent file.
type BlockDownloadHandler struct {
	pp.NoopHandler
	pp.NoopBep3Handler
	pp.NoopBep6Handler

	Info         metainfo.Info                              // Required
	OnBlock      func(index, offset uint32, b []byte) error // Required
	RequestBlock func(c *pp.PeerConn) error                 // Required
}

// NewBlockDownloadHandler returns a new BlockDownloadHandler.
func NewBlockDownloadHandler(info metainfo.Info,
	onBlock func(pieceIndex, pieceOffset uint32, b []byte) error,
	requestBlock func(c *pp.PeerConn) error) BlockDownloadHandler {
	return BlockDownloadHandler{
		Info:         info,
		OnBlock:      onBlock,
		RequestBlock: requestBlock,
	}
}

// OnHandShake implements the interface Handler#OnHandShake.
//
// Notice: it uses the field Data to store the inner data, you mustn't override
// it.
func (fd BlockDownloadHandler) OnHandShake(c *pp.PeerConn) (err error) {
	if err = c.SetUnchoked(); err == nil {
		err = c.SetInterested()
	}
	return
}

/// ---------------------------------------------------------------------------
/// BEP 3

func (fd BlockDownloadHandler) request(pc *pp.PeerConn) (err error) {
	if pc.PeerChoked {
		err = pp.ErrChoked
	} else {
		err = fd.RequestBlock(pc)
	}
	return
}

// Piece implements the interface Bep3Handler#Piece.
func (fd BlockDownloadHandler) Piece(c *pp.PeerConn, i, b uint32, p []byte) (err error) {
	if err = fd.OnBlock(i, b, p); err == nil {
		err = fd.request(c)
	}
	return
}

// Unchoke implements the interface Bep3Handler#Unchoke.
func (fd BlockDownloadHandler) Unchoke(pc *pp.PeerConn) (err error) {
	return fd.request(pc)
}

// Have implements the interface Bep3Handler#Have.
func (fd BlockDownloadHandler) Have(pc *pp.PeerConn, index uint32) (err error) {
	pc.BitField.Set(index)
	return
}

/// ---------------------------------------------------------------------------
/// BEP 6

// HaveAll implements the interface Bep6Handler#HaveAll.
func (fd BlockDownloadHandler) HaveAll(pc *pp.PeerConn) (err error) {
	pc.BitField = pp.NewBitField(fd.Info.CountPieces(), true)
	return
}

// Reject implements the interface Bep6Handler#Reject.
func (fd BlockDownloadHandler) Reject(pc *pp.PeerConn, index, begin, length uint32) (err error) {
	pc.BitField.Unset(index)
	return fd.request(pc)
}
