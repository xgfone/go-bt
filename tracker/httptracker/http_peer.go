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

package httptracker

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"

	"github.com/xgfone/bt/bencode"
	"github.com/xgfone/bt/metainfo"
)

var errInvalidPeer = errors.New("invalid peer information format")

// Peer is a tracker peer.
type Peer struct {
	// ID is the peer's self-selected ID.
	ID string `bencode:"peer id"` // BEP 3

	// IP is the IP address or dns name.
	IP   string `bencode:"ip"`   // BEP 3
	Port uint16 `bencode:"port"` // BEP 3
}

// Addresses returns the list of the addresses that the peer listens on.
func (p Peer) Addresses() (addrs []metainfo.Address, err error) {
	if ip := net.ParseIP(p.IP); len(ip) != 0 {
		return []metainfo.Address{{IP: &net.IPAddr{IP: ip}, Port: p.Port}}, nil
	}

	ips, err := net.LookupIP(p.IP)
	if _len := len(ips); err == nil && len(ips) != 0 {
		addrs = make([]metainfo.Address, _len)
		for i, ip := range ips {
			addrs[i] = metainfo.Address{IP: &net.IPAddr{IP: ip}, Port: p.Port}
		}
	}

	return
}

// Peers is a set of the peers.
type Peers []Peer

// UnmarshalBencode implements the interface bencode.Unmarshaler.
func (ps *Peers) UnmarshalBencode(b []byte) (err error) {
	var v interface{}
	if err = bencode.DecodeBytes(b, &v); err != nil {
		return
	}

	switch vs := v.(type) {
	case string: // BEP 23
		_len := len(vs)
		if _len%6 != 0 {
			return metainfo.ErrInvalidAddr
		}

		peers := make(Peers, 0, _len/6)
		for i := 0; i < _len; i += 6 {
			var addr metainfo.Address
			if err = addr.UnmarshalBinary([]byte(vs[i : i+6])); err != nil {
				return
			}
			peers = append(peers, Peer{IP: addr.IP.String(), Port: addr.Port})
		}

		*ps = peers
	case []interface{}: // BEP 3
		peers := make(Peers, len(vs))
		for i, p := range vs {
			m, ok := p.(map[string]interface{})
			if !ok {
				return errInvalidPeer
			}

			pid, ok := m["peer id"].(string)
			if !ok {
				return errInvalidPeer
			}

			ip, ok := m["ip"].(string)
			if !ok {
				return errInvalidPeer
			}

			port, ok := m["port"].(int64)
			if !ok {
				return errInvalidPeer
			}

			peers[i] = Peer{ID: pid, IP: ip, Port: uint16(port)}
		}
		*ps = peers
	default:
		return errInvalidPeer
	}
	return
}

// MarshalBencode implements the interface bencode.Marshaler.
func (ps Peers) MarshalBencode() (b []byte, err error) {
	for _, p := range ps {
		if p.ID == "" {
			return ps.marshalCompactBencode() // BEP 23
		}
	}

	// BEP 3
	buf := bytes.NewBuffer(make([]byte, 0, 64*len(ps)))
	buf.WriteByte('l')
	for _, p := range ps {
		if err = bencode.NewEncoder(buf).Encode(p); err != nil {
			return
		}
	}
	buf.WriteByte('e')
	b = buf.Bytes()
	return
}

func (ps Peers) marshalCompactBencode() (b []byte, err error) {
	buf := bytes.NewBuffer(make([]byte, 0, 6*len(ps)))
	for _, peer := range ps {
		ip := net.ParseIP(peer.IP).To4()
		if len(ip) == 0 {
			return nil, errInvalidPeer
		}
		buf.Write(ip[:])
		binary.Write(buf, binary.BigEndian, peer.Port)
	}
	return bencode.EncodeBytes(buf.Bytes())
}

// Peers6 is a set of the peers for IPv6 in the compact case.
//
// BEP 7
type Peers6 []Peer

// UnmarshalBencode implements the interface bencode.Unmarshaler.
func (ps *Peers6) UnmarshalBencode(b []byte) (err error) {
	var s string
	if err = bencode.DecodeBytes(b, &s); err != nil {
		return
	}

	_len := len(s)
	if _len%18 != 0 {
		return metainfo.ErrInvalidAddr
	}

	peers := make(Peers6, 0, _len/18)
	for i := 0; i < _len; i += 18 {
		var addr metainfo.Address
		if err = addr.UnmarshalBinary([]byte(s[i : i+18])); err != nil {
			return
		}
		peers = append(peers, Peer{IP: addr.IP.String(), Port: addr.Port})
	}

	*ps = peers
	return
}

// MarshalBencode implements the interface bencode.Marshaler.
func (ps Peers6) MarshalBencode() (b []byte, err error) {
	buf := bytes.NewBuffer(make([]byte, 0, 18*len(ps)))
	for _, peer := range ps {
		ip := net.ParseIP(peer.IP).To16()
		if len(ip) == 0 {
			return nil, errInvalidPeer
		}

		buf.Write(ip[:])
		binary.Write(buf, binary.BigEndian, peer.Port)
	}
	return bencode.EncodeBytes(buf.Bytes())
}
