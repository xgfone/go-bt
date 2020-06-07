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
	"bytes"
	"fmt"
	"io"
	"net"

	"github.com/xgfone/bt/metainfo"
)

// Node represents a node information.
type Node struct {
	ID   metainfo.Hash
	Addr metainfo.Address
}

// NewNode returns a new Node.
func NewNode(id metainfo.Hash, ip net.IP, port int) Node {
	return Node{ID: id, Addr: metainfo.NewAddress(ip, uint16(port))}
}

// NewNodeByUDPAddr returns a new Node with the id and the UDP address.
func NewNodeByUDPAddr(id metainfo.Hash, addr *net.UDPAddr) (n Node) {
	n.ID = id
	n.Addr.FromUDPAddr(addr)
	return
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

// MarshalBinary implements the interface binary.BinaryMarshaler.
func (n Node) MarshalBinary() (data []byte, err error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(48)
	if _, err = n.WriteBinary(buf); err == nil {
		data = buf.Bytes()
	}
	return
}

// UnmarshalBinary implements the interface binary.BinaryUnmarshaler.
func (n *Node) UnmarshalBinary(b []byte) error {
	if len(b) < 26 {
		return io.ErrShortBuffer
	}

	copy(n.ID[:], b[:20])
	return n.Addr.UnmarshalBinary(b[20:])
}
