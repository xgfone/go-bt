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

import (
	"sort"

	"github.com/xgfone/bt/metainfo"
	pp "github.com/xgfone/bt/peerprotocol"
)

// PieceBlock is the information of the block in the piece.
type PieceBlock struct {
	Index  uint32 // The index of the piece
	Offset uint32 // The offset from the beginning of the piece
	Length uint32 // The length of the block, which is equal to 2^14 in general
}

// pieceIndexes is a set of the piece indexes.
type pieceIndexes []uint32

func (ps pieceIndexes) Len() int           { return len(ps) }
func (ps pieceIndexes) Less(i, j int) bool { return ps[i] < ps[j] }
func (ps pieceIndexes) Swap(i, j int)      { ps[i], ps[j] = ps[j], ps[i] }

// Sort sorts itself.
func (ps pieceIndexes) Sort() { sort.Sort(ps) }

// Contains reports whether it contains the given index.
func (ps pieceIndexes) Contains(index uint32) bool {
	for _, p := range ps {
		if p == index {
			return true
		}
	}
	return false
}

// pieceIndexInfo is the information of the piece indexes in the remote peers.
type pieceIndexInfo struct {
	Fasts    pieceIndexes
	Pieces   pieceIndexes
	Suggests pieceIndexes
}

var _ pp.Handler = &FileDownloadHandler{}
var _ pp.Bep3Handler = &FileDownloadHandler{}
var _ pp.Bep6Handler = &FileDownloadHandler{}

// FileDownloadHandler is used to downloads the files in the torrent file.
type FileDownloadHandler struct {
	pp.NoopHandler
	pp.NoopBep3Handler
	pp.NoopBep6Handler

	Writer *metainfo.Writer // Required

	Logger       func(format string, args ...interface{})            // Optional
	GetNextBlock func(fasts, suggests, pieces []uint32) []PieceBlock // Required
}

// NewFileDownloadHandler returns a new FileDownloadHandler.
func NewFileDownloadHandler(writer *metainfo.Writer,
	getNextBlock func([]uint32, []uint32, []uint32) []PieceBlock,
	log func(string, ...interface{})) *FileDownloadHandler {
	return &FileDownloadHandler{
		Logger:       log,
		Writer:       writer,
		GetNextBlock: getNextBlock,
	}
}

// OnHandShake implements the interface Handler#OnHandShake.
//
// Notice: it uses the field Data to store the inner data, you mustn't override
// it.
func (fd *FileDownloadHandler) OnHandShake(c *pp.PeerConn) (err error) {
	c.Data = &pieceIndexInfo{Pieces: make([]uint32, 0, fd.Writer.Info.CountPieces())}
	return
}

// OnMessage implements the interface Handler#OnMessage.
func (fd *FileDownloadHandler) OnMessage(c *pp.PeerConn, m pp.Message) error {
	if log := fd.Logger; log != nil {
		log("unexpected message '%s' from '%s'", m.Type, c.RemoteAddr().String())
	}
	return nil
}

// RequestPiece sends the Request to get the piece data.
func (fd *FileDownloadHandler) RequestPiece(c *pp.PeerConn) (err error) {
	pi := c.Data.(*pieceIndexInfo)
	blocks := fd.GetNextBlock(pi.Fasts, pi.Suggests, pi.Pieces)
	for _, b := range blocks {
		if err = c.SendRequest(b.Index, b.Offset, b.Length); err != nil {
			break
		}
	}
	return
}

/// ---------------------------------------------------------------------------
/// BEP 3

// Piece implements the interface Bep3Handler#Piece.
func (fd *FileDownloadHandler) Piece(c *pp.PeerConn, i, b uint32, p []byte) error {
	_, err := fd.Writer.WriteBlock(p, i, b)
	return err
}

// Unchoke implements the interface Bep3Handler#Unchoke.
func (fd *FileDownloadHandler) Unchoke(pc *pp.PeerConn) (err error) {
	return fd.RequestPiece(pc)
}

// Have implements the interface Bep3Handler#Have.
func (fd *FileDownloadHandler) Have(pc *pp.PeerConn, index uint32) (err error) {
	pi := pc.Data.(*pieceIndexInfo)
	if !pi.Pieces.Contains(index) {
		pi.Pieces = append(pi.Pieces, index)
		pi.Pieces.Sort()
	}
	return
}

// Bitfield implements the interface Bep3Handler#Bitfield.
func (fd *FileDownloadHandler) Bitfield(pc *pp.PeerConn, bits []bool) (err error) {
	pi := pc.Data.(*pieceIndexInfo)
	for i, t := range bits {
		if t {
			pi.Pieces = append(pi.Pieces, uint32(i))
		}
	}
	return
}

/// ---------------------------------------------------------------------------
/// BEP 6

// HaveAll implements the interface Bep6Handler#HaveAll.
func (fd *FileDownloadHandler) HaveAll(pc *pp.PeerConn) (err error) {
	pi := pc.Data.(*pieceIndexInfo)
	for i := range fd.Writer.Info.Pieces {
		pi.Pieces = append(pi.Pieces, uint32(i))
	}
	return
}

// Suggest implements the interface Bep6Handler#Suggest.
func (fd *FileDownloadHandler) Suggest(pc *pp.PeerConn, index uint32) (err error) {
	pi := pc.Data.(*pieceIndexInfo)
	if !pi.Suggests.Contains(index) {
		pi.Suggests = append(pi.Suggests, index)
		pi.Suggests.Sort()
	}
	return
}

// AllowedFast implements the interface Bep6Handler#AllowedFast.
func (fd *FileDownloadHandler) AllowedFast(pc *pp.PeerConn, index uint32) (err error) {
	pi := pc.Data.(*pieceIndexInfo)
	if !pi.Fasts.Contains(index) {
		pi.Fasts = append(pi.Fasts, index)
		pi.Fasts.Sort()
	}
	return
}

// Reject implements the interface Bep6Handler#Reject.
func (fd *FileDownloadHandler) Reject(pc *pp.PeerConn, index, begin, length uint32) (err error) {
	pi := pc.Data.(*pieceIndexInfo)
	pi.Suggests = fd.removePiece(pi.Suggests, index)
	pi.Pieces = fd.removePiece(pi.Pieces, index)
	return
}

func (fd *FileDownloadHandler) removePiece(ps pieceIndexes, index uint32) pieceIndexes {
	for i, p := range ps {
		if p == index {
			copy(ps[i:], ps[i+1:])
			return ps[:len(ps)-1]
		}
	}
	return ps
}
