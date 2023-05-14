// Copyright 2020 xgfone, 2023 idk
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

func TestFilePieces_Merge(t *testing.T) {
	fps := FilePieces{
		{
			Index:  1,
			Offset: 100,
			Length: 100,
		},
		{
			Index:  1,
			Offset: 400,
			Length: 100,
		},
		{
			Index:  2,
			Offset: 500,
			Length: 100,
		},
		{
			Index:  2,
			Offset: 700,
			Length: 100,
		},
		{
			Index:  1,
			Offset: 500,
			Length: 100,
		},
		{
			Index:  2,
			Offset: 200,
			Length: 100,
		},
		{
			Index:  2,
			Offset: 300,
			Length: 100,
		},
		{
			Index:  1,
			Offset: 300,
			Length: 100,
		},
	}

	fps = fps.Merge()

	if len(fps) != 5 {
		t.Error(fps)
	} else if fps[0].Index != 1 || fps[0].Offset != 100 || fps[0].Length != 100 {
		t.Error(fps[0])
	} else if fps[1].Index != 1 || fps[1].Offset != 300 || fps[1].Length != 300 {
		t.Error(fps[1])
	} else if fps[2].Index != 2 || fps[2].Offset != 200 || fps[2].Length != 200 {
		t.Error(fps[2])
	} else if fps[3].Index != 2 || fps[3].Offset != 500 || fps[3].Length != 100 {
		t.Error(fps[3])
	} else if fps[4].Index != 2 || fps[4].Offset != 700 || fps[4].Length != 100 {
		t.Error(fps[4])
	}
}
