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
	"io"
	"os"
)

// Reader is used to read the data referred by the torrent file.
type Reader struct {
	Info  Info
	Root  string
	files map[string]*os.File
}

// NewReader returns a new Reader.
func NewReader(rootDir string, info Info) *Reader {
	return &Reader{
		Root:  rootDir,
		Info:  info,
		files: make(map[string]*os.File, len(info.Files)),
	}
}

// Close implements the interface io.Closer to closes the opened files.
func (r *Reader) Close() error {
	for name, file := range r.files {
		if file != nil {
			file.Close()
			delete(r.files, name)
		}
	}
	return nil
}

// ReadBlock reads a data block.
func (r *Reader) ReadBlock(p []byte, pieceIndex, pieceOffset uint32) (int, error) {
	return r.ReadAt(p, r.Info.PieceOffset(pieceIndex, pieceOffset))
}

// ReadAt implements the interface io.ReaderAt.
func (r *Reader) ReadAt(p []byte, offset int64) (n int, err error) {
	var m int
	var f *os.File

	for _len := len(p); n < _len; {
		file, fileOffset := r.Info.GetFileByOffset(offset)
		if file.Length == 0 {
			break
		}

		length := int(file.Length-fileOffset) + n
		if _len < length {
			length = _len
		} else if length <= n {
			err = io.EOF
			break
		}

		filename := file.PathWithPrefix(r.Root, r.Info)
		if f, err = r.open(filename); err != nil {
			break
		}

		m, err = f.ReadAt(p[n:length], fileOffset)
		n += m
		offset += int64(m)
		if err != nil {
			break
		}
	}

	return
}

func (r *Reader) open(filename string) (f *os.File, err error) {
	if r.files == nil {
		r.files = make(map[string]*os.File, len(r.Info.Files))
	}

	f, ok := r.files[filename]
	if !ok {
		if f, err = os.Open(filename); err == nil {
			r.files[filename] = f
		}
	}
	return
}
