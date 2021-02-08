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

// getAllInfoFiles returns the list of the all files in a certain directory
// recursively.
func getAllInfoFiles(rootDir string) (files []File, err error) {
	err = filepath.Walk(rootDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return fmt.Errorf("error getting relative path: %s", err)
		}

		paths := strings.Split(relPath, string(filepath.Separator))
		for _, name := range paths {
			if name == ".git" {
				return nil
			}
		}

		files = append(files, File{
			Paths:  paths,
			Length: fi.Size(),
		})

		return nil
	})

	if err == nil && len(files) == 0 {
		err = fmt.Errorf("no files in the directory '%s'", rootDir)
	}

	return
}

// NewInfoFromFilePath returns a new Info from a file or directory.
func NewInfoFromFilePath(root string, pieceLength int64) (info Info, err error) {
	root = filepath.Clean(root)

	fi, err := os.Stat(root)
	if err != nil {
		return
	} else if !fi.IsDir() {
		info.Length = fi.Size()
	} else if info.Files, err = getAllInfoFiles(root); err != nil {
		return
	}

	sort.Sort(files(info.Files))
	info.Pieces, err = GeneratePiecesFromFiles(info.AllFiles(), pieceLength,
		func(file File) (io.ReadCloser, error) {
			if _len := len(file.Paths); _len > 0 {
				paths := make([]string, 0, _len+1)
				paths = append(paths, root)
				paths = append(paths, file.Paths...)
				return os.Open(filepath.Join(paths...))
			}
			return os.Open(root)
		})

	if err == nil {
		info.Name = filepath.Base(root)
		info.PieceLength = pieceLength
	} else {
		err = fmt.Errorf("error generating pieces: %s", err)
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

// PieceOffset returns the total offset of the piece.
//
// offset is the offset relative to the beginning of the piece.
func (info Info) PieceOffset(index, offset uint32) int64 {
	return int64(index)*info.PieceLength + int64(offset)
}

// GetFileByOffset returns the file and its offset by the total offset.
//
// If fileOffset is eqaul to file.Length, it means to reach the end.
func (info Info) GetFileByOffset(offset int64) (file File, fileOffset int64) {
	if !info.IsDir() {
		if offset > info.Length {
			panic(fmt.Errorf("offset '%d' exceeds the maximum length '%d'",
				offset, info.Length))
		}
		return File{Length: info.Length, Paths: []string{info.Name}}, offset
	}

	fileOffset = offset
	for i, _len := 0, len(info.Files)-1; i <= _len; i++ {
		file = info.Files[i]
		if fileOffset < file.Length {
			return
		} else if fileOffset == file.Length && i == _len {
			return
		}
		fileOffset -= file.Length
	}

	if fileOffset > file.Length {
		panic(fmt.Errorf("offset '%d' exceeds the maximum length '%d'",
			offset, info.TotalLength()))
	}

	return
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
