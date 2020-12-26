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
	"bytes"
	"os"
	"testing"
)

func generateTestPieceData(len int, numData byte) []byte {
	b := make([]byte, len)
	for i := 0; i < len; i++ {
		b[i] = numData
	}
	return b
}

func TestWriterAndReader(t *testing.T) {
	info := Info{
		Name:        "test_rw",
		PieceLength: 64,
		Files: []File{
			{Length: 100, Paths: []string{"file1"}},
			{Length: 200, Paths: []string{"file2"}},
			{Length: 300, Paths: []string{"file3"}},
		},
	}

	r := NewReader("", info)
	w := NewWriter("", info, 0600)
	defer func() {
		w.Close()
		r.Close()
		os.RemoveAll("test_rw")
	}()

	datalen := 600
	wdata := make([]byte, datalen)
	for i := 0; i < datalen; i++ {
		wdata[i] = byte(i%26 + 97)
	}

	n, err := w.WriteAt(wdata, 0)
	if err != nil {
		t.Error(err)
		return
	} else if n != len(wdata) {
		t.Errorf("expect wrote '%d', but got '%d'\n", len(wdata), n)
		return
	}

	rdata := make([]byte, datalen)
	n, err = r.ReadAt(rdata, 0)
	if err != nil {
		t.Error(err)
		return
	} else if n != len(rdata) {
		t.Errorf("expect read '%d', but got '%d'\n", len(rdata), n)
		return
	}

	if bytes.Compare(rdata, wdata) != 0 {
		t.Errorf("expect read '%x', but got '%x'\n", wdata, rdata)
	}
}
