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

import "testing"

func TestInfo_GetFileByOffset(t *testing.T) {
	FileInfo := Info{
		Name:        "test_rw",
		PieceLength: 64,
		Length:      600,
	}
	file, fileOffset := FileInfo.GetFileByOffset(0)
	if file.Offset(FileInfo) != 0 || fileOffset != 0 {
		t.Errorf("expect fileOffset='%d', but got '%d'", 0, fileOffset)
	}
	file, fileOffset = FileInfo.GetFileByOffset(100)
	if file.Offset(FileInfo) != 0 || fileOffset != 100 {
		t.Errorf("expect fileOffset='%d', but got '%d'", 100, fileOffset)
	}
	file, fileOffset = FileInfo.GetFileByOffset(600)
	if file.Offset(FileInfo) != 0 || fileOffset != 600 {
		t.Errorf("expect fileOffset='%d', but got '%d'", 600, fileOffset)
	}

	DirInfo := Info{
		Name:        "test_rw",
		PieceLength: 64,
		Files: []File{
			{Length: 100, Paths: []string{"file1"}},
			{Length: 200, Paths: []string{"file2"}},
			{Length: 300, Paths: []string{"file3"}},
		},
	}
	file, fileOffset = DirInfo.GetFileByOffset(0)
	if file.Offset(DirInfo) != 0 || file.Length != 100 || fileOffset != 0 {
		t.Errorf("expect fileOffset='%d', but got '%d'", 0, fileOffset)
	}
	file, fileOffset = DirInfo.GetFileByOffset(50)
	if file.Offset(DirInfo) != 0 || file.Length != 100 || fileOffset != 50 {
		t.Errorf("expect fileOffset='%d', but got '%d'", 50, fileOffset)
	}
	file, fileOffset = DirInfo.GetFileByOffset(100)
	if file.Offset(DirInfo) != 100 || file.Length != 200 || fileOffset != 0 {
		t.Errorf("expect fileOffset='%d', but got '%d'", 0, fileOffset)
	}
	file, fileOffset = DirInfo.GetFileByOffset(200)
	if file.Offset(DirInfo) != 100 || file.Length != 200 || fileOffset != 100 {
		t.Errorf("expect fileOffset='%d', but got '%d'", 100, fileOffset)
	}
	file, fileOffset = DirInfo.GetFileByOffset(300)
	if file.Offset(DirInfo) != 300 || file.Length != 300 || fileOffset != 0 {
		t.Errorf("expect fileOffset='%d', but got '%d'", 0, fileOffset)
	}
	file, fileOffset = DirInfo.GetFileByOffset(400)
	if file.Offset(DirInfo) != 300 || file.Length != 300 || fileOffset != 100 {
		t.Errorf("expect fileOffset='%d', but got '%d'", 100, fileOffset)
	}
	file, fileOffset = DirInfo.GetFileByOffset(600)
	if file.Offset(DirInfo) != 300 || file.Length != 300 || fileOffset != 300 {
		t.Errorf("expect fileOffset='%d', but got '%d'", 300, fileOffset)
	}
}

func TestNewInfoFromFilePath(t *testing.T) {
	info, err := NewInfoFromFilePath("info.go", PieceSize256KB)
	if err != nil {
		t.Error(err)
	} else if info.Name != "info.go" || info.Files != nil {
		t.Errorf("invalid info %+v\n", info)
	}

	info, err = NewInfoFromFilePath("../metainfo", PieceSize256KB)
	if err != nil {
		t.Error(err)
	} else if info.Name != "metainfo" || info.Files == nil || info.Length > 0 {
		t.Errorf("invalid info %+v\n", info)
	}

	info, err = NewInfoFromFilePath("../../bt", PieceSize256KB)
	if err != nil {
		t.Error(err)
	} else if info.Name != "bt" || info.Files == nil || info.Length > 0 {
		t.Errorf("invalid info %+v\n", info)
	}
}
