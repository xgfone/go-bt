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
	"path/filepath"
)

// Writer is used to write the data referred by the torrent file.
type Writer interface {
	io.Closer
	io.WriterAt

	Info() Info
	WriteBlock(pieceIndex, pieceOffset uint32, p []byte) (int, error)
}

const wflag = os.O_WRONLY | os.O_CREATE

type writer struct {
	info  Info
	root  string
	mode  os.FileMode
	files map[string]*os.File
}

// NewWriter returns a new Writer.
//
// If fileMode is equal to 0, it is 0600 by default.
//
// Notice: fileMode is only used when writing the data.
func NewWriter(rootDir string, info Info, fileMode os.FileMode) Writer {
	if fileMode == 0 {
		fileMode = 0600
	}

	return &writer{
		root:  rootDir,
		info:  info,
		mode:  fileMode,
		files: make(map[string]*os.File, len(info.Files)),
	}
}

func (w *writer) open(filename string) (f *os.File, err error) {
	if w.files == nil {
		w.files = make(map[string]*os.File, len(w.info.Files))
	}

	f, ok := w.files[filename]
	if !ok {
		if err = os.MkdirAll(filepath.Dir(filename), 0700); err == nil {
			if f, err = os.OpenFile(filename, wflag, w.mode); err == nil {
				w.files[filename] = f
			}
		}
	}

	return
}

func (w *writer) Info() Info { return w.info }

// Close implements the interface io.Closer to closes the opened files.
func (w *writer) Close() error {
	for name, file := range w.files {
		if file != nil {
			file.Close()
			delete(w.files, name)
		}
	}
	return nil
}

// WriteBlock writes a data block.
func (w *writer) WriteBlock(pieceIndex, pieceOffset uint32, p []byte) (int, error) {
	return w.WriteAt(p, w.info.PieceOffset(pieceIndex, pieceOffset))
}

// WriteAt implements the interface io.WriterAt.
func (w *writer) WriteAt(p []byte, offset int64) (n int, err error) {
	var m int
	var f *os.File

	for _len := len(p); n < _len; {
		file, fileOffset := w.info.GetFileByOffset(offset)
		if file.Length == 0 || file.Length == fileOffset {
			err = io.ErrShortWrite
			break
		}

		length := int(file.Length-fileOffset) + n
		if _len < length {
			length = _len
		} else if length <= n {
			break
		}

		filename := file.PathWithPrefix(w.root, w.info)
		if f, err = w.open(filename); err != nil {
			break
		}

		m, err = f.WriteAt(p[n:length], fileOffset)
		n += m
		offset += int64(m)
		if err != nil {
			break
		}

		f.Sync()
	}

	return
}
