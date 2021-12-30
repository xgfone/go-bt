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

// Package udptracker implements the tracker protocol based on UDP.
//
// You can use the package to implement a UDP tracker server to track the
// information that other peers upload or download the file, or to create
// a UDP tracker client to communicate with the UDP tracker server.
package udptracker

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/xgfone/bt/metainfo"
)

// ProtocolID is magic constant for the udp tracker connection.
//
// BEP 15
const ProtocolID = uint64(0x41727101980)

// Predefine some actions.
//
// BEP 15
const (
	ActionConnect  = uint32(0)
	ActionAnnounce = uint32(1)
	ActionScrape   = uint32(2)
	ActionError    = uint32(3)
)

// AnnounceRequest represents the announce request used by UDP tracker.
//
// BEP 15
type AnnounceRequest struct {
	InfoHash metainfo.Hash
	PeerID   metainfo.Hash

	Downloaded int64
	Left       int64
	Uploaded   int64
	Event      uint32

	IP      net.IP
	Key     int32
	NumWant int32 // -1 for default
	Port    uint16

	Exts []Extension // BEP 41
}

// DecodeFrom decodes the request from b.
func (r *AnnounceRequest) DecodeFrom(b []byte, ipv4 bool) {
	r.InfoHash = metainfo.NewHash(b[0:20])
	r.PeerID = metainfo.NewHash(b[20:40])
	r.Downloaded = int64(binary.BigEndian.Uint64(b[40:48]))
	r.Left = int64(binary.BigEndian.Uint64(b[48:56]))
	r.Uploaded = int64(binary.BigEndian.Uint64(b[56:64]))
	r.Event = binary.BigEndian.Uint32(b[64:68])

	if ipv4 {
		r.IP = make(net.IP, net.IPv4len)
		copy(r.IP, b[68:72])
		b = b[72:]
	} else {
		r.IP = make(net.IP, net.IPv6len)
		copy(r.IP, b[68:84])
		b = b[84:]
	}

	r.Key = int32(binary.BigEndian.Uint32(b[0:4]))
	r.NumWant = int32(binary.BigEndian.Uint32(b[4:8]))
	r.Port = binary.BigEndian.Uint16(b[8:10])

	b = b[10:]
	for len(b) > 0 {
		var ext Extension
		parsed := ext.DecodeFrom(b)
		r.Exts = append(r.Exts, ext)
		b = b[parsed:]
	}
}

// EncodeTo encodes the request to buf.
func (r AnnounceRequest) EncodeTo(buf *bytes.Buffer) {
	buf.Grow(82)
	buf.Write(r.InfoHash[:])
	buf.Write(r.PeerID[:])

	binary.Write(buf, binary.BigEndian, r.Downloaded)
	binary.Write(buf, binary.BigEndian, r.Left)
	binary.Write(buf, binary.BigEndian, r.Uploaded)
	binary.Write(buf, binary.BigEndian, r.Event)

	if ip := r.IP.To4(); ip != nil {
		buf.Write(ip[:])
	} else {
		buf.Write(r.IP[:])
	}

	binary.Write(buf, binary.BigEndian, r.Key)
	binary.Write(buf, binary.BigEndian, r.NumWant)
	binary.Write(buf, binary.BigEndian, r.Port)

	for _, ext := range r.Exts {
		ext.EncodeTo(buf)
	}
}

// AnnounceResponse represents the announce response used by UDP tracker.
//
// BEP 15
type AnnounceResponse struct {
	Interval  uint32
	Leechers  uint32
	Seeders   uint32
	Addresses []metainfo.Address
}

// EncodeTo encodes the response to buf.
func (r AnnounceResponse) EncodeTo(buf *bytes.Buffer) {
	buf.Grow(12 + len(r.Addresses)*18)
	binary.Write(buf, binary.BigEndian, r.Interval)
	binary.Write(buf, binary.BigEndian, r.Leechers)
	binary.Write(buf, binary.BigEndian, r.Seeders)
	for _, addr := range r.Addresses {
		if ip := addr.To4(); ip != nil {
			buf.Write(ip.IP[:])
		} else if ip := addr.To16(); ip != nil {
			buf.Write(ip.IP[:])
		}
		binary.Write(buf, binary.BigEndian, addr.Port)
	}
}

// DecodeFrom decodes the response from b.
func (r *AnnounceResponse) DecodeFrom(b []byte, ipv4 bool) {
	r.Interval = binary.BigEndian.Uint32(b[:4])
	r.Leechers = binary.BigEndian.Uint32(b[4:8])
	r.Seeders = binary.BigEndian.Uint32(b[8:12])

	b = b[12:]
	iplen := net.IPv6len
	if ipv4 {
		iplen = net.IPv4len
	}

	_len := len(b)
	step := iplen + 2
	r.Addresses = make([]metainfo.Address, 0, _len/step)
	for i := step; i <= _len; i += step {
		ip := make(net.IP, iplen)
		copy(ip, b[i-step:i-2])
		port := binary.BigEndian.Uint16(b[i-2 : i])
		r.Addresses = append(r.Addresses, metainfo.Address{IP: &net.IPAddr{IP: ip}, Port: port})
	}
}

// ScrapeResponse represents the UDP SCRAPE response.
//
// BEP 15
type ScrapeResponse struct {
	Seeders   uint32
	Leechers  uint32
	Completed uint32
}

// EncodeTo encodes the response to buf.
func (r ScrapeResponse) EncodeTo(buf *bytes.Buffer) {
	binary.Write(buf, binary.BigEndian, r.Seeders)
	binary.Write(buf, binary.BigEndian, r.Completed)
	binary.Write(buf, binary.BigEndian, r.Leechers)
}

// DecodeFrom decodes the response from b.
func (r *ScrapeResponse) DecodeFrom(b []byte) {
	r.Seeders = binary.BigEndian.Uint32(b[:4])
	r.Completed = binary.BigEndian.Uint32(b[4:8])
	r.Leechers = binary.BigEndian.Uint32(b[8:12])
}

// Predefine some UDP extension types.
//
// BEP 41
const (
	EndOfOptions ExtensionType = iota
	Nop
	URLData
)

// NewEndOfOptions returns a new EndOfOptions UDP extension.
func NewEndOfOptions() Extension { return Extension{Type: EndOfOptions} }

// NewNop returns a new Nop UDP extension.
func NewNop() Extension { return Extension{Type: Nop} }

// NewURLData returns a new URLData UDP extension.
func NewURLData(data []byte) Extension {
	return Extension{Type: URLData, Length: uint8(len(data)), Data: data}
}

// ExtensionType represents the type of UDP extension.
type ExtensionType uint8

func (et ExtensionType) String() string {
	switch et {
	case EndOfOptions:
		return "EndOfOptions"
	case Nop:
		return "Nop"
	case URLData:
		return "URLData"
	default:
		return fmt.Sprintf("ExtensionType(%d)", et)
	}
}

// Extension represent the extension used by the UDP ANNOUNCE request.
//
// BEP 41
type Extension struct {
	Type   ExtensionType
	Length uint8
	Data   []byte
}

// EncodeTo encodes the response to buf.
func (e Extension) EncodeTo(buf *bytes.Buffer) {
	if _len := uint8(len(e.Data)); e.Length == 0 && _len != 0 {
		e.Length = _len
	} else if _len != e.Length {
		panic("the length of data is inconsistent")
	}

	buf.WriteByte(byte(e.Type))
	buf.WriteByte(e.Length)
	buf.Write(e.Data)
}

// DecodeFrom decodes the response from b.
func (e *Extension) DecodeFrom(b []byte) (parsed int) {
	switch len(b) {
	case 0:
	case 1:
		e.Type = ExtensionType(b[0])
		parsed = 1
	default:
		e.Type = ExtensionType(b[0])
		e.Length = b[1]
		parsed = 2
		if e.Length > 0 {
			parsed += int(e.Length)
			e.Data = b[2:parsed]
		}
	}
	return
}
