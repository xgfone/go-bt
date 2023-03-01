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
	"testing"

	"github.com/xgfone/bt/bencode"
)

func TestAddress(t *testing.T) {
	addrs := []HostAddr{
		{Host: "1.2.3.4", Port: 123},
		{Host: "www.example.com", Port: 456},
	}
	expect := `ll7:1.2.3.4i123eel15:www.example.comi456eee`

	if result, err := bencode.EncodeString(addrs); err != nil {
		t.Error(err)
	} else if result != expect {
		t.Errorf("expect %s, but got %s\n", expect, result)
	}

	var raddrs []HostAddr
	if err := bencode.DecodeString(expect, &raddrs); err != nil {
		t.Error(err)
	} else if len(raddrs) != len(addrs) {
		t.Errorf("expect addrs length %d, but got %d\n", len(addrs), len(raddrs))
	} else {
		for i, addr := range addrs {
			if !addr.Equal(raddrs[i]) {
				t.Errorf("%d: expect %v, but got %v\n", i, addr, raddrs[i])
			}
		}
	}
}
