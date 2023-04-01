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

package downloader

import pp "github.com/xgfone/go-bt/peerprotocol"

// BlockDownloadHandler is used to downloads the files in the torrent file.
type BlockDownloadHandler struct {
	pp.NoopHandler
	pp.NoopBep3Handler
	pp.NoopBep6Handler

	OnBlock  func(index, offset uint32, b []byte) error
	ReqBlock func(c *pp.PeerConn) error
	PieceNum int
}

// NewBlockDownloadHandler returns a new BlockDownloadHandler.
func NewBlockDownloadHandler(pieceNum int, reqBlock func(c *pp.PeerConn) error,
	onBlock func(pieceIndex, pieceOffset uint32, b []byte) error) BlockDownloadHandler {
	return BlockDownloadHandler{
		OnBlock:  onBlock,
		ReqBlock: reqBlock,
		PieceNum: pieceNum,
	}
}

/// ---------------------------------------------------------------------------
/// BEP 3

func (fd BlockDownloadHandler) request(pc *pp.PeerConn) (err error) {
	if fd.ReqBlock == nil {
		return nil
	}

	if pc.PeerChoked {
		err = pp.ErrChoked
	} else {
		err = fd.ReqBlock(pc)
	}
	return
}

// Piece implements the interface Bep3Handler#Piece.
func (fd BlockDownloadHandler) Piece(c *pp.PeerConn, i, b uint32, p []byte) error {
	if fd.OnBlock != nil {
		if err := fd.OnBlock(i, b, p); err != nil {
			return err
		}
	}
	return fd.request(c)
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
	if fd.PieceNum > 0 {
		pc.BitField = pp.NewBitField(fd.PieceNum, true)
	}
	return
}

// Reject implements the interface Bep6Handler#Reject.
func (fd BlockDownloadHandler) Reject(pc *pp.PeerConn, index, begin, length uint32) (err error) {
	pc.BitField.Unset(index)
	return fd.request(pc)
}
