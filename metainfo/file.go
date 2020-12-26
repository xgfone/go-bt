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
	"path/filepath"
)

// File represents a file in the multi-file case.
type File struct {
	// Length is the length of the file in bytes.
	Length int64 `json:"length" bencode:"length"` // BEP 3

	// Paths is a list containing one or more string elements that together
	// represent the path and filename. Each element in the list corresponds
	// to either a directory name or (in the case of the final element) the
	// filename.
	//
	// For example, a the file "dir1/dir2/file.ext" would consist of three
	// string elements: "dir1", "dir2", and "file.ext". This is encoded as
	// a bencoded list of strings such as l4:dir14:dir28:file.exte.
	Paths []string `json:"path" bencode:"path"` // BEP 3
}

func (f File) String() string {
	return filepath.Join(f.Paths...)
}

// Path returns the path of the current.
func (f File) Path(info Info) string {
	if info.IsDir() {
		paths := make([]string, len(f.Paths)+1)
		paths[0] = info.Name
		copy(paths[1:], f.Paths)
		return filepath.Join(paths...)
	}
	return info.Name
}

// PathWithPrefix returns the path of the current with the prefix directory.
func (f File) PathWithPrefix(prefix string, info Info) string {
	if info.IsDir() {
		paths := make([]string, len(f.Paths)+2)
		paths[0] = prefix
		paths[1] = info.Name
		copy(paths[2:], f.Paths)
		return filepath.Join(paths...)
	}
	return filepath.Join(prefix, info.Name)
}

// Offset returns the offset of the current file from the start.
func (f File) Offset(info Info) (ret int64) {
	path := f.Path(info)
	for _, file := range info.AllFiles() {
		if path == file.Path(info) {
			return
		}
		ret += file.Length
	}
	panic("not found")
}

type files []File

func (fs files) Len() int           { return len(fs) }
func (fs files) Less(i, j int) bool { return fs[i].String() < fs[j].String() }
func (fs files) Swap(i, j int)      { f := fs[i]; fs[i] = fs[j]; fs[j] = f }

// FilePieces returns the information of the pieces referred by the file.
func (f File) FilePieces(info Info) (fps FilePieces) {
	if f.Length < 1 {
		return nil
	}

	startOffset := f.Offset(info)
	startPieceIndex := startOffset / info.PieceLength
	startPieceOffset := startOffset % info.PieceLength

	endOffset := startOffset + f.Length
	endPieceIndex := endOffset / info.PieceLength
	endPieceOffset := endOffset % info.PieceLength

	if startPieceIndex == endPieceIndex {
		return FilePieces{{
			Index:  uint32(startPieceIndex),
			Offset: uint32(startPieceOffset),
			Length: uint32(endPieceOffset - startPieceOffset),
		}}
	}

	fps = make(FilePieces, 0, endPieceIndex-startPieceIndex+1)
	fps = append(fps, FilePiece{
		Index:  uint32(startPieceIndex),
		Offset: uint32(startPieceOffset),
		Length: uint32(info.PieceLength - startPieceOffset),
	})
	for i := startPieceIndex + 1; i < endPieceIndex; i++ {
		fps = append(fps, FilePiece{Index: uint32(i), Length: uint32(info.PieceLength)})
	}
	fps = append(fps, FilePiece{Index: uint32(endPieceIndex), Length: uint32(endPieceOffset)})
	return
}
