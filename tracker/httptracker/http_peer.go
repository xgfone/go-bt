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
	"errors"
	"net"

	"github.com/xgfone/bt/bencode"
	"github.com/xgfone/bt/metainfo"
)

var errInvalidPeer = errors.New("invalid bt peer information format")

// Peer is a tracker peer.
type Peer struct {
	ID   string `bencode:"peer id"` // BEP 3, the peer's self-selected ID.
	IP   string `bencode:"ip"`      // BEP 3, an IP address or dns name.
	Port uint16 `bencode:"port"`    // BEP 3
}

var (
	_ bencode.Marshaler   = new(Peers)
	_ bencode.Unmarshaler = new(Peers)
)

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
		var addrs metainfo.CompactIPv4Addrs
		if err = addrs.UnmarshalBinary([]byte(vs)); err != nil {
			return err
		}

		peers := make(Peers, len(addrs))
		for i, addr := range addrs {
			peers[i] = Peer{IP: addr.IP.String(), Port: addr.Port}
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
	// BEP 23
	if b, err = ps.marshalCompactBencode(); err == nil {
		return
	}

	// BEP 3
	return bencode.EncodeBytes([]Peer(ps))
}

func (ps Peers) marshalCompactBencode() (b []byte, err error) {
	addrs := make(metainfo.CompactIPv4Addrs, len(ps))
	for i, p := range ps {
		ip := net.ParseIP(p.IP).To4()
		if ip == nil {
			return nil, errInvalidPeer
		}
		addrs[i] = metainfo.CompactAddr{IP: ip, Port: p.Port}
	}
	return addrs.MarshalBencode()
}

var (
	_ bencode.Marshaler   = new(Peers6)
	_ bencode.Unmarshaler = new(Peers6)
)

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

	var addrs metainfo.CompactIPv6Addrs
	if err = addrs.UnmarshalBinary([]byte(s)); err != nil {
		return err
	}

	peers := make(Peers6, len(addrs))
	for i, addr := range addrs {
		peers[i] = Peer{IP: addr.IP.String(), Port: addr.Port}
	}
	*ps = peers

	return
}

// MarshalBencode implements the interface bencode.Marshaler.
func (ps Peers6) MarshalBencode() (b []byte, err error) {
	addrs := make(metainfo.CompactIPv6Addrs, len(ps))
	for i, p := range ps {
		ip := net.ParseIP(p.IP)
		if ip == nil {
			return nil, errInvalidPeer
		}
		addrs[i] = metainfo.CompactAddr{IP: ip, Port: p.Port}
	}
	return addrs.MarshalBencode()
}
