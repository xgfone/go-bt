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
	"bytes"
	"log"
	"testing"

	"github.com/eyedeekay/sam3"
)

func TestCompactIP(t *testing.T) {
	ipv4 := CompactIP([]byte{1, 2, 3, 4})
	b, err := ipv4.MarshalBencode()
	if err != nil {
		t.Fatal(err)
	}

	log.Println("IPv4 Test", ipv4.String(), len(ipv4))

	var ip CompactIP
	if err = ip.UnmarshalBencode(b); err != nil {
		t.Error(err, ip)
	} else if ip.String() != "1.2.3.4" {
		t.Error(ip.String(), ",", ip)
	}
}

func TestCompactIP6(t *testing.T) {
	ipv6 := CompactIP([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	b, err := ipv6.MarshalBencode()
	if err != nil {
		t.Fatal(err)
	}

	log.Println("IPv6 Test", ipv6.String(), len(ipv6))

	var ip CompactIP
	if err = ip.UnmarshalBencode(b); err != nil {
		t.Error(err)
	} else if ip.String() != "102:304:506:708:90a:b0c:d0e:f10" {
		t.Error(ip.String(), ",", ip)
	}
}

func TestCompactI2P(t *testing.T) {
	sam, err := sam3.NewSAM("127.0.0.1:7656")
	if err != nil {
		t.Fatal(err)
	}
	defer sam.Close()
	i2pkeys, err := sam.NewKeys()
	if err != nil {
		t.Fatal(err)
	}
	dh := i2pkeys.Address.DestHash()
	i2p := CompactIP(dh[:])
	b, err := i2p.MarshalBencode()
	if err != nil {
		t.Fatal(err)
	}

	log.Println("I2P Test", i2p.String(), len(b))

	var ip CompactIP
	if err = ip.UnmarshalBencode(b); err != nil {
		t.Error(err)
	} else if ip.String() != dh.String() {
		t.Error(ip, dh)
	}
}

func TestUtMetadataExtendedMsg(t *testing.T) {
	buf := new(bytes.Buffer)
	data := []byte{0x31, 0x32, 0x33, 0x34, 0x35}
	m1 := UtMetadataExtendedMsg{MsgType: 1, Piece: 2, TotalSize: 1024, Data: data}
	if err := m1.EncodeToPayload(buf); err != nil {
		t.Fatal(err)
	}

	msg := Message{Type: MTypeExtended, ExtendedPayload: buf.Bytes()}
	m2, err := msg.UtMetadataExtendedMsg()
	if err != nil {
		t.Fatal(err)
	} else if m2.MsgType != 1 || m2.Piece != 2 || m2.TotalSize != 1024 {
		t.Error(m2)
	} else if !bytes.Equal(m2.Data, data) {
		t.Fail()
	}
}
