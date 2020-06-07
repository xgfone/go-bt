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

import (
	"bytes"
	"testing"
)

func TestCompactIP(t *testing.T) {
	ipv4 := CompactIP([]byte{1, 2, 3, 4})
	b, err := ipv4.MarshalBencode()
	if err != nil {
		t.Fatal(err)
	}

	var ip CompactIP
	if err = ip.UnmarshalBencode(b); err != nil {
		t.Error(err)
	} else if ip.String() != "1.2.3.4" {
		t.Error(ip)
	}
}

func TestUtMetadataExtendedMsg(t *testing.T) {
	buf := new(bytes.Buffer)
	data := []byte{0x31, 0x32, 0x33, 0x34, 0x35}
	m1 := UtMetadataExtendedMsg{MsgType: 1, Piece: 2, TotalSize: 1024, Data: data}
	if err := m1.EncodeToPayload(buf); err != nil {
		t.Fatal(err)
	}

	msg := Message{Type: Extended, ExtendedPayload: buf.Bytes()}
	m2, err := msg.UtMetadataExtendedMsg()
	if err != nil {
		t.Fatal(err)
	} else if m2.MsgType != 1 || m2.Piece != 2 || m2.TotalSize != 1024 {
		t.Error(m2)
	} else if !bytes.Equal(m2.Data, data) {
		t.Fail()
	}
}
