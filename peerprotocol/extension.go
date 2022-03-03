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
	"errors"
	"log"
	"net"

	"github.com/eyedeekay/sam3/i2pkeys"
	"github.com/xgfone/bt/bencode"
)

var errInvalidIP = errors.New("invalid ipv4 or ipv6")

// Predefine some extended message identifiers.
const (
	ExtendedIDHandshake = 0 // BEP 10
)

// Predefine some extended message names.
const (
	ExtendedMessageNameMetadata = "ut_metadata" // BEP 9
	ExtendedMessageNamePex      = "ut_pex"      // BEP 11
)

// Predefine some "ut_metadata" extended message types.
const (
	UtMetadataExtendedMsgTypeRequest = 0 // BEP 9
	UtMetadataExtendedMsgTypeData    = 1 // BEP 9
	UtMetadataExtendedMsgTypeReject  = 2 // BEP 9
)

// CompactIP is used to handle the compact ipv4 or ipv6.
type CompactIP []byte

func (ci CompactIP) String() string {
	//return net.IP(ci).String()
	switch len(ci) {
	case net.IPv4len:
		return net.IP(ci).String()
	case net.IPv6len:
		return net.IP(ci).String()
	case 32:
		return i2pkeys.I2PAddr(ci).DestHash().String()
	}
	log.Printf("CI String: %b, %d", ci, len(ci))
	return ""
}

// MarshalBencode implements the interface bencode.Marshaler.
func (ci CompactIP) MarshalBencode() ([]byte, error) {
	log.Println("Marshal Bytes:", ci, len(ci))
	if len(ci) == net.IPv4len {
		log.Println("Marshal IPv4 Bytes:", ci, len(ci))
		ip := []byte(ci)
		ic, err := bencode.EncodeBytes(ip[:])
		if err != nil {
			return nil, err
		}
		return ic, nil
	}
	if len(ci) == net.IPv6len {
		log.Println("Marshal IPv6 Bytes:", ci, len(ci))
		ip := []byte(ci)
		ic, err := bencode.EncodeBytes(ip[:])
		if err != nil {
			return nil, err
		}
		return ic, nil
	}
	if len(ci) == 32 {
		log.Println("Marshal I2P Bytes:", ci, len(ci))
		i2p := i2pkeys.I2PAddr(ci).DestHash()
		return bencode.EncodeBytes(i2p[:])
	}
	return nil, errInvalidIP
}

// UnmarshalBencode implements the interface bencode.Unmarshaler.
func (ci *CompactIP) UnmarshalBencode(b []byte) (err error) {
	log.Println("Unmarshal Bytes:", b, len(b))
	if len(b) >= net.IPv4len && len(b) < net.IPv6len {
		log.Println("Unmarshal IPv4 Bytes:", b, len(b))
		ip := net.IP(b[len(b)-net.IPv4len:])
		if ipv4 := ip.To4(); len(ipv4) != 0 {
			ip = ipv4
		}
		*ci = CompactIP(ip[:])
		log.Println("Unmarshal IPv4:", ip, len(ip))
		return nil
	}
	if len(b) >= net.IPv6len && len(b) < 32 {
		log.Println("Unmarshal IPv6 Bytes:", b, len(b))
		ip := net.IP(b[len(b)-net.IPv6len:])
		if ipv6 := ip.To16(); len(ipv6) != 0 {
			ip = ipv6
		}
		*ci = CompactIP(ip[:])
		log.Println("Unmarshal IPv6:", ip, len(ip))
		return nil
	}
	if len(b) >= 32 {
		log.Println("Unmarshal I2P Bytes:", b, len(b)-32)
		i2p := i2pkeys.I2PAddr(b[len(b)-32:]).DestHash()
		*ci = i2p[:]
		log.Println("Unmarshal I2P:", i2p, len(i2p))
		return nil
	}
	return errInvalidIP
}

// ExtendedHandshakeMsg represent the extended handshake message.
//
// BEP 10
type ExtendedHandshakeMsg struct {
	// M is the type of map[ExtendedMessageName]ExtendedMessageID.
	M    map[string]uint8 `bencode:"m"`              // BEP 10
	V    string           `bencode:"v,omitempty"`    // BEP 10
	Reqq int              `bencode:"reqq,omitempty"` // BEP 10

	// Port is the local client port, which is redundant and no need
	// for the receiving side of the connection to send this.
	Port   uint16    `bencode:"p,omitempty"`      // BEP 10
	IPv6   []byte    `bencode:"ipv6,omitempty"`   // BEP 10
	IPv4   CompactIP `bencode:"ipv4,omitempty"`   // BEP 10
	YourIP CompactIP `bencode:"yourip,omitempty"` // BEP 10

	MetadataSize int `bencode:"metadata_size,omitempty"` // BEP 9
}

// Decode decodes the extended handshake message from b.
func (ehm *ExtendedHandshakeMsg) Decode(b []byte) (err error) {
	return bencode.DecodeBytes(b, ehm)
}

// Encode encodes the extended handshake message to b.
func (ehm ExtendedHandshakeMsg) Encode() (b []byte, err error) {
	buf := bytes.NewBuffer(make([]byte, 0, 128))
	if err = bencode.NewEncoder(buf).Encode(ehm); err != nil {
		b = buf.Bytes()
	}
	return
}

// UtMetadataExtendedMsg represents the "ut_metadata" extended message.
type UtMetadataExtendedMsg struct {
	MsgType uint8 `bencode:"msg_type"` // BEP 9
	Piece   int   `bencode:"piece"`    // BEP 9

	// They are only used by "data" type
	TotalSize int    `bencode:"total_size,omitempty"` // BEP 9
	Data      []byte `bencode:"-"`
}

// EncodeToPayload encodes UtMetadataExtendedMsg to extended payload
// and write the result into buf.
func (um UtMetadataExtendedMsg) EncodeToPayload(buf *bytes.Buffer) (err error) {
	if um.MsgType != UtMetadataExtendedMsgTypeData {
		um.TotalSize = 0
		um.Data = nil
	}

	buf.Grow(len(um.Data) + 50)
	if err = bencode.NewEncoder(buf).Encode(um); err == nil {
		_, err = buf.Write(um.Data)
	}
	return
}

// EncodeToBytes is equal to
//
//   buf := new(bytes.Buffer)
//   err = um.EncodeToPayload(buf)
//   return buf.Bytes(), err
//
func (um UtMetadataExtendedMsg) EncodeToBytes() (b []byte, err error) {
	buf := bytes.NewBuffer(make([]byte, 0, 128))
	if err = um.EncodeToPayload(buf); err == nil {
		b = buf.Bytes()
	}
	return
}

// DecodeFromPayload decodes the extended payload to itself.
func (um *UtMetadataExtendedMsg) DecodeFromPayload(b []byte) (err error) {
	dec := bencode.NewDecoder(bytes.NewReader(b))
	if err = dec.Decode(&um); err == nil {
		um.Data = b[dec.BytesParsed():]
	}
	return
}
