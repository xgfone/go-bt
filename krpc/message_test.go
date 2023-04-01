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

package krpc

import (
	"testing"

	"github.com/xgfone/go-bt/bencode"
)

func TestMessage(t *testing.T) {
	s, err := bencode.EncodeString(Message{RO: true})
	if err != nil {
		t.Fatal(err)
	}

	var ms map[string]interface{}
	if err := bencode.DecodeString(s, &ms); err != nil {
		t.Fatal(err)
	} else if len(ms) != 3 {
		t.Fatal()
	} else if v, ok := ms["t"]; !ok || v != "" {
		t.Fatal()
	} else if v, ok := ms["y"]; !ok || v != "" {
		t.Fatal()
	} else if v, ok := ms["ro"].(int64); !ok || v != 1 {
		t.Fatal()
	}
}
