// Copyright 2020~2023 xgfone
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
	"bytes"
	"encoding"
	"fmt"
	"io"

	"github.com/xgfone/bt/bencode"
	"github.com/xgfone/bt/metainfo"
)

// Node represents a node information.
type Node struct {
	ID   metainfo.Hash
	Addr Addr
}

// NewNode returns a new Node.
func NewNode(id metainfo.Hash, addr Addr) Node {
	return Node{ID: id, Addr: addr}
}

func (n Node) String() string {
	return fmt.Sprintf("Node<%x@%s>", n.ID, n.Addr)
}

// Equal reports whether n is equal to o.
func (n Node) Equal(o Node) bool {
	return n.ID == o.ID && n.Addr.Equal(o.Addr)
}

// WriteBinary is the same as MarshalBinary, but writes the result into w
// instead of returning.
func (n Node) WriteBinary(w io.Writer) (m int, err error) {
	var n1, n2 int
	if n1, err = w.Write(n.ID[:]); err == nil {
		m = n1
		if n2, err = n.Addr.WriteBinary(w); err == nil {
			m += n2
		}
	}
	return
}

var (
	_ encoding.BinaryMarshaler   = new(Node)
	_ encoding.BinaryUnmarshaler = new(Node)
)

// MarshalBinary implements the interface encoding.BinaryMarshaler,
// which implements "Compact node info".
//
// See http://bittorrent.org/beps/bep_0005.html.
func (n Node) MarshalBinary() (data []byte, err error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(40)
	if _, err = n.WriteBinary(buf); err == nil {
		data = buf.Bytes()
	}
	return
}

// UnmarshalBinary implements the interface encoding.BinaryUnmarshaler,
// which implements "Compact node info".
//
// See http://bittorrent.org/beps/bep_0005.html.
func (n *Node) UnmarshalBinary(b []byte) error {
	if len(b) < 26 {
		return io.ErrShortBuffer
	}

	copy(n.ID[:], b[:20])
	return n.Addr.UnmarshalBinary(b[20:])
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

var (
	_ bencode.Marshaler          = new(CompactIPv4Nodes)
	_ bencode.Unmarshaler        = new(CompactIPv4Nodes)
	_ encoding.BinaryMarshaler   = new(CompactIPv4Nodes)
	_ encoding.BinaryUnmarshaler = new(CompactIPv4Nodes)
)

// CompactIPv4Nodes is a set of IPv4 Nodes.
type CompactIPv4Nodes []Node

// MarshalBinary implements the interface encoding.BinaryMarshaler.
func (cns CompactIPv4Nodes) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(26 * len(cns))
	for _, ni := range cns {
		if ni.Addr.IP = ni.Addr.IP.To4(); len(ni.Addr.IP) == 0 {
			continue
		}
		if n, err := ni.WriteBinary(buf); err != nil {
			return nil, err
		} else if n != 26 {
			panic(fmt.Errorf("CompactIPv4Nodes: the invalid node info length '%d'", n))
		}
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary implements the interface encoding.BinaryUnmarshaler.
func (cns *CompactIPv4Nodes) UnmarshalBinary(b []byte) (err error) {
	_len := len(b)
	if _len%26 != 0 {
		return fmt.Errorf("CompactIPv4Nodes: invalid node info length '%d'", _len)
	}

	nis := make([]Node, 0, _len/26)
	for i := 0; i < _len; i += 26 {
		var ni Node
		if err = ni.UnmarshalBinary(b[i : i+26]); err != nil {
			return
		}
		nis = append(nis, ni)
	}

	*cns = nis
	return
}

// MarshalBencode implements the interface bencode.Marshaler.
func (cns CompactIPv4Nodes) MarshalBencode() (b []byte, err error) {
	if b, err = cns.MarshalBinary(); err == nil {
		b, err = bencode.EncodeBytes(b)
	}
	return
}

// UnmarshalBencode implements the interface bencode.Unmarshaler.
func (cns *CompactIPv4Nodes) UnmarshalBencode(b []byte) (err error) {
	var data []byte
	if err = bencode.DecodeBytes(b, &data); err == nil {
		err = cns.UnmarshalBinary(data)
	}
	return
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

var (
	_ bencode.Marshaler          = new(CompactIPv6Nodes)
	_ bencode.Unmarshaler        = new(CompactIPv6Nodes)
	_ encoding.BinaryMarshaler   = new(CompactIPv6Nodes)
	_ encoding.BinaryUnmarshaler = new(CompactIPv6Nodes)
)

// CompactIPv6Nodes is a set of IPv6 Nodes.
type CompactIPv6Nodes []Node

// MarshalBinary implements the interface encoding.BinaryMarshaler.
func (cns CompactIPv6Nodes) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(38 * len(cns))
	for _, ni := range cns {
		ni.Addr.IP = ni.Addr.IP.To16()
		if n, err := ni.WriteBinary(buf); err != nil {
			return nil, err
		} else if n != 38 {
			panic(fmt.Errorf("CompactIPv6Nodes: the invalid node info length '%d'", n))
		}
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary implements the interface encoding.BinaryUnmarshaler.
func (cns *CompactIPv6Nodes) UnmarshalBinary(b []byte) (err error) {
	_len := len(b)
	if _len%38 != 0 {
		return fmt.Errorf("CompactIPv6Nodes: invalid node info length '%d'", _len)
	}

	nis := make([]Node, 0, _len/38)
	for i := 0; i < _len; i += 38 {
		var ni Node
		if err = ni.UnmarshalBinary(b[i : i+38]); err != nil {
			return
		}
		nis = append(nis, ni)
	}

	*cns = nis
	return
}

// MarshalBencode implements the interface bencode.Marshaler.
func (cns CompactIPv6Nodes) MarshalBencode() (b []byte, err error) {
	if b, err = cns.MarshalBinary(); err == nil {
		b, err = bencode.EncodeBytes(b)
	}
	return
}

// UnmarshalBencode implements the interface bencode.Unmarshaler.
func (cns *CompactIPv6Nodes) UnmarshalBencode(b []byte) (err error) {
	var data []byte
	if err = bencode.DecodeBytes(b, &data); err == nil {
		err = cns.UnmarshalBinary(data)
	}
	return
}
