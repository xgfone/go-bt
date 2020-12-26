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

func TestHash(t *testing.T) {
	hexHash := "0001020304050607080909080706050403020100"

	b, err := NewHashFromHexString(hexHash).MarshalBencode()
	if err != nil {
		t.Fatal(err)
	}

	var h Hash
	if err = h.UnmarshalBencode(b); err != nil {
		t.Fatal(err)
	}

	if hexs := h.String(); hexs != hexHash {
		t.Errorf("expect '%s', but got '%s'\n", hexHash, hexs)
	}

	h = Hash{}
	hexHash = "0001020304050607080900010203040506070809"
	err = h.UnmarshalBinary([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
	if err != nil {
		t.Error(err)
	} else if hexs := h.HexString(); hexs != hexHash {
		t.Errorf("expect '%s', but got '%s'\n", hexHash, hexs)
	}
}

func TestHashes(t *testing.T) {
	hexHash1 := "0101010101010101010101010101010101010101"
	hexHash2 := "0202020202020202020202020202020202020202"

	hashes := Hashes{
		NewHashFromHexString(hexHash1),
		NewHashFromHexString(hexHash2),
	}

	b, err := hashes.MarshalBencode()
	if err != nil {
		t.Fatal(err)
	}

	hashes = Hashes{}
	if err = hashes.UnmarshalBencode(b); err != nil {
		t.Fatal(err)
	}

	if _len := len(hashes); _len != 2 {
		t.Fatalf("expect the len(hashes)==2, but got '%d'", _len)
	}

	for i, h := range hashes {
		if i == 0 {
			if hexs := h.HexString(); hexs != hexHash1 {
				t.Errorf("index %d: expect '%s', but got '%s'\n", i, hexHash1, hexs)
			}
		} else {
			if hexs := h.HexString(); hexs != hexHash2 {
				t.Errorf("index %d: expect '%s', but got '%s'\n", i, hexHash2, hexs)
			}
		}
	}
}
