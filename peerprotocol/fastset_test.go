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

package peerprotocol

import (
	"net"
	"testing"

	"github.com/eyedeekay/go-i2p-bt/metainfo"
)

func TestGenerateAllowedFastSet(t *testing.T) {
	hexs := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	infohash := metainfo.NewHashFromHexString(hexs)

	sets := make([]uint32, 7)
	GenerateAllowedFastSet(sets, 1313, net.ParseIP("80.4.4.200"), infohash)
	for i, v := range sets {
		switch i {
		case 0:
			if v != 1059 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		case 1:
			if v != 431 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		case 2:
			if v != 808 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		case 3:
			if v != 1217 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		case 4:
			if v != 287 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		case 5:
			if v != 376 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		case 6:
			if v != 1188 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		default:
			t.Errorf("unknown '%d' at the index '%d'", v, i)
		}
	}

	sets = make([]uint32, 9)
	GenerateAllowedFastSet(sets, 1313, net.ParseIP("80.4.4.200"), infohash)
	for i, v := range sets {
		switch i {
		case 0:
			if v != 1059 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		case 1:
			if v != 431 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		case 2:
			if v != 808 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		case 3:
			if v != 1217 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		case 4:
			if v != 287 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		case 5:
			if v != 376 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		case 6:
			if v != 1188 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		case 7:
			if v != 353 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		case 8:
			if v != 508 {
				t.Errorf("unknown '%d' at the index '%d'", v, i)
			}
		default:
			t.Errorf("unknown '%d' at the index '%d'", v, i)
		}
	}
}
