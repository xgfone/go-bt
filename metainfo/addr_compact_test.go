// Copyright 2023 xgfone
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
	"net"
	"testing"

	"github.com/xgfone/bt/bencode"
)

func TestCompactAddr(t *testing.T) {
	addrs := []CompactAddr{
		{IP: net.ParseIP("172.16.1.1").To4(), Port: 123},
		{IP: net.ParseIP("192.168.1.1").To4(), Port: 456},
	}
	expect := "l6:\xac\x10\x01\x01\x00\x7b6:\xc0\xa8\x01\x01\x01\xc8e"

	buf := new(bytes.Buffer)
	if err := bencode.NewEncoder(buf).Encode(addrs); err != nil {
		t.Error(err)
	} else if result := buf.String(); result != expect {
		t.Errorf("expect %s, but got %x\n", expect, result)
	}

	var raddrs []CompactAddr
	if err := bencode.DecodeString(expect, &raddrs); err != nil {
		t.Error(err)
	} else if len(raddrs) != len(addrs) {
		t.Errorf("expect addrs length %d, but got %d\n", len(addrs), len(raddrs))
	} else {
		for i, addr := range addrs {
			if !addr.Equal(raddrs[i]) {
				t.Errorf("%d: expect addr %v, but got %v\n", i, addr, raddrs[i])
			}
		}
	}
}
