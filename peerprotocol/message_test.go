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

package peerprotocol

import "testing"

func TestBitField(t *testing.T) {
	bf := NewBitFieldFromBools([]bool{
		false, true, false, true, false, true, true, true,
		false, false, true, false, false, true, true, false})
	if bf.IsSet(0) {
		t.Error(0)
	} else if !bf.IsSet(1) {
		t.Error(1)
	} else if !bf.IsSet(7) {
		t.Error(7)
	} else if bf.IsSet(8) {
		t.Error(8)
	} else if bf.IsSet(15) {
		t.Error(15)
	}

	bf.Set(9)
	if !bf.IsSet(9) {
		t.Error(9)
	}

	bf.Unset(10)
	if bf.IsSet(10) {
		t.Error(10)
	}

	bs := bf.Bools()
	if len(bs) != 16 {
		t.Fatal(bs)
	} else if !bs[9] {
		t.Error(9)
	} else if bs[10] {
		t.Error(10)
	}
}
