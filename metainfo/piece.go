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

package metainfo

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/xgfone/bt/utils"
)

// Piece represents a torrent file piece.
type Piece struct {
	info  Info
	index int
}

// Piece returns the Piece by the index starting with 0.
func (info Info) Piece(index int) Piece {
	if n := len(info.Pieces); index >= n {
		panic(fmt.Errorf("Info.Piece: index '%d' exceeds maximum '%d'", index, n))
	}
	return Piece{info: info, index: index}
}

// Index returns the index of the current piece.
func (p Piece) Index() int { return p.index }

// Offset returns the offset that the current piece is in all the files.
func (p Piece) Offset() int64 { return int64(p.index) * p.info.PieceLength }

// Hash returns the hash representation of the piece.
func (p Piece) Hash() (h Hash) { return p.info.Pieces[p.index] }

// Length returns the length of the current piece.
func (p Piece) Length() int64 {
	if p.index == p.info.CountPieces()-1 {
		return p.info.TotalLength() - int64(p.index)*p.info.PieceLength
	}
	return p.info.PieceLength
}

// GeneratePieces generates the pieces from the reader.
func GeneratePieces(r io.Reader, pieceLength int64) (hs Hashes, err error) {
	buf := make([]byte, pieceLength)
	for {
		h := sha1.New()
		written, err := utils.CopyNBuffer(h, r, pieceLength, buf)
		if written > 0 {
			hs = append(hs, NewHash(h.Sum(nil)))
		}

		if err == io.EOF {
			return hs, nil
		}

		if err != nil {
			return nil, err
		}
	}
}

func writeFiles(w io.Writer, files []File, open func(File) (io.ReadCloser, error)) error {
	buf := make([]byte, 8192)
	for _, file := range files {
		r, err := open(file)
		if err != nil {
			return fmt.Errorf("error opening %s: %s", file, err)
		}

		n, err := utils.CopyNBuffer(w, r, file.Length, buf)
		r.Close()

		if n != file.Length {
			return fmt.Errorf("error copying %s: %s", file, err)
		}
	}
	return nil
}

// GeneratePiecesFromFiles generates the pieces from the files.
func GeneratePiecesFromFiles(files []File, pieceLength int64,
	open func(File) (io.ReadCloser, error)) (Hashes, error) {
	if pieceLength <= 0 {
		return nil, errors.New("piece length must be a positive integer")
	}

	pr, pw := io.Pipe()
	defer pr.Close()

	go func() { pw.CloseWithError(writeFiles(pw, files, open)) }()
	return GeneratePieces(pr, pieceLength)
}

// PieceBlock represents a block in a piece.
type PieceBlock struct {
	Index  uint32 // The index of the piece.
	Offset uint32 // The offset from the beginning of the piece.
	Length uint32 // The length of the block, which is equal to 2^14 in general.
}

// PieceBlocks is a set of PieceBlocks.
type PieceBlocks []PieceBlock

func (pbs PieceBlocks) Len() int      { return len(pbs) }
func (pbs PieceBlocks) Swap(i, j int) { pbs[i], pbs[j] = pbs[j], pbs[i] }
func (pbs PieceBlocks) Less(i, j int) bool {
	if pbs[i].Index < pbs[j].Index {
		return true
	} else if pbs[i].Index == pbs[j].Index && pbs[i].Offset < pbs[j].Offset {
		return true
	}
	return false
}

// FilePiece represents the piece range used by a file, which is used to
// calculate the downloaded piece when downloading the file.
type FilePiece struct {
	// The index of the current piece.
	Index uint32

	// The offset bytes from the beginning of the current piece,
	// which is equal to 0 in general.
	Offset uint32

	// The length of the data, which is equal to PieceLength in Info in general.
	// For most implementations, PieceLength is equal to 2^18.
	// So, a piece can contain sixteen blocks.
	Length uint32
}

// TotalOffset return the total offset from the beginning of all the files.
func (fp FilePiece) TotalOffset(pieceLength int64) int64 {
	return int64(fp.Index)*pieceLength + int64(fp.Offset)
}

// Blocks returns the lists of the blocks of the piece.
func (fp FilePiece) Blocks() PieceBlocks {
	bs := make(PieceBlocks, 0, 16)
	for offset, rest := fp.Offset, fp.Length; rest > 0; {
		length := uint32(16384)
		if rest < length {
			length = rest
		}

		bs = append(bs, PieceBlock{Index: fp.Index, Offset: offset, Length: length})
		offset += length
		rest -= length
	}
	return bs
}

// FilePieces is a set of FilePieces.
type FilePieces []FilePiece

func (fps FilePieces) Len() int      { return len(fps) }
func (fps FilePieces) Swap(i, j int) { fps[i], fps[j] = fps[j], fps[i] }
func (fps FilePieces) Less(i, j int) bool {
	if fps[i].Index < fps[j].Index {
		return true
	} else if fps[i].Index == fps[j].Index && fps[i].Offset < fps[j].Offset {
		return true
	}
	return false
}

// Merge merges the contiguous pieces to the one piece.
func (fps FilePieces) Merge() FilePieces {
	_len := len(fps)
	if _len < 2 {
		return fps
	}
	sort.Sort(fps)
	results := make(FilePieces, 0, _len)

	lastpos := 0
	curindex := fps[0].Index
	for i, fp := range fps {
		if fp.Index != curindex {
			results = fps.merge(fps[lastpos:i], results)
			lastpos = i
			curindex = fp.Index
		}
	}

	if lastpos < _len {
		results = fps.merge(fps[lastpos:_len], results)
	}

	return results
}

func (fps FilePieces) merge(fpset, results FilePieces) FilePieces {
	switch _len := len(fpset); _len {
	case 0:
	case 1:
		results = append(results, fpset[0])
	default:
		index := fpset[0].Index
		offset := fpset[0].Offset
		length := fpset[0].Length

		var last int
		for i := 1; i < _len; i++ {
			fp := fpset[i]

			if offset+length == fp.Offset {
				length += fp.Length
				continue
			}

			last = i
			results = append(results, FilePiece{
				Index:  index,
				Offset: offset,
				Length: length,
			})

			offset = fp.Offset
			length = fp.Length
		}

		if last < _len {
			results = append(results, FilePiece{
				Index:  index,
				Offset: offset,
				Length: length,
			})
		}
	}

	return results
}
