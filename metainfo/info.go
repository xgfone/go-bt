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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Info is the file inforatino.
type Info struct {
	// Name is the name of the file in the single file case.
	// Or, it is the name of the directory in the muliple file case.
	Name string `json:"name" bencode:"name"` // BEP 3

	// PieceLength is the number of bytes in each piece, which is usually
	// a power of 2.
	PieceLength int64 `json:"piece length" bencode:"piece length"` // BEP 3

	// Pieces is the concatenation of all 20-byte SHA1 hash values,
	// one per piece (byte string, i.e. not urlencoded).
	Pieces Hashes `json:"pieces" bencode:"pieces"` // BEP 3

	// Length is the length of the file in bytes in the single file case.
	//
	// It's mutually exclusive with Files.
	Length int64 `json:"length,omitempty" bencode:"length,omitempty"` // BEP 3

	// Files is the list of all the files in the multi-file case.
	//
	// For the purposes of the other keys, the multi-file case is treated
	// as only having a single file by concatenating the files in the order
	// they appear in the files list.
	//
	// It's mutually exclusive with Length.
	Files []File `json:"files,omitempty" bencode:"files,omitempty"` // BEP 3
}

// NewInfoFromFilePath returns a new Info from a file or directory.
func NewInfoFromFilePath(root string, pieceLength int64) (info Info, err error) {
	info.Name = filepath.Base(root)
	info.PieceLength = pieceLength
	err = filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == root && !fi.IsDir() { // The root is a file.
			info.Length = fi.Size()
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("error getting relative path: %s", err)
		}

		info.Files = append(info.Files, File{
			Paths:  strings.Split(relPath, string(filepath.Separator)),
			Length: fi.Size(),
		})

		return nil
	})

	if err == nil {
		sort.Sort(files(info.Files))
		info.Pieces, err = GeneratePiecesFromFiles(info.AllFiles(), info.PieceLength,
			func(file File) (io.ReadCloser, error) {
				if _len := len(file.Paths); _len > 0 {
					paths := make([]string, 0, _len+1)
					paths = append(paths, root)
					paths = append(paths, file.Paths...)
					return os.Open(filepath.Join(paths...))
				}
				return os.Open(root)
			})

		if err != nil {
			err = fmt.Errorf("error generating pieces: %s", err)
		}
	}

	return
}

// IsDir reports whether the name is a directory, that's, the file is not
// a single file.
func (info Info) IsDir() bool { return len(info.Files) != 0 }

// CountPieces returns the number of the pieces.
func (info Info) CountPieces() int { return len(info.Pieces) }

// TotalLength returns the total length of the torrent file.
func (info Info) TotalLength() (ret int64) {
	if info.IsDir() {
		for _, fi := range info.Files {
			ret += fi.Length
		}
	} else {
		ret = info.Length
	}
	return
}

// Piece returns the Piece by the index starting with 0.
func (info Info) Piece(index int) Piece {
	if n := len(info.Pieces); index >= n {
		panic(fmt.Errorf("Info.Piece: index '%d' exceeds maximum '%d'", index, n))
	}
	return Piece{info: info, index: index}
}

// AllFiles returns all the files.
//
// Notice: for the single file, the Path is nil.
func (info Info) AllFiles() []File {
	if info.IsDir() {
		return info.Files
	}
	return []File{{Length: info.Length}}
}
