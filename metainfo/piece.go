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

	"github.com/xgfone/bt/utils"
)

// Piece represents a torrent file piece.
type Piece struct {
	info  Info
	index int
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
