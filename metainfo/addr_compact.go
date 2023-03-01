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
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/xgfone/bt/bencode"
)

// CompactAddr represents an address based on ip and port,
// which implements "Compact IP-address/port info".
//
// See http://bittorrent.org/beps/bep_0005.html.
type CompactAddr struct {
	IP   net.IP // For IPv4, its length must be 4.
	Port uint16
}

// NewCompactAddr returns a new compact Addr with ip and port.
func NewCompactAddr(ip net.IP, port uint16) CompactAddr {
	return CompactAddr{IP: ip, Port: port}
}

// NewCompactAddrFromUDPAddr converts *net.UDPAddr to a new CompactAddr.
func NewCompactAddrFromUDPAddr(ua *net.UDPAddr) CompactAddr {
	return CompactAddr{IP: ua.IP, Port: uint16(ua.Port)}
}

// Valid reports whether the addr is valid.
func (a CompactAddr) Valid() bool {
	return len(a.IP) > 0 && a.Port > 0
}

// Equal reports whether a is equal to o.
func (a CompactAddr) Equal(o CompactAddr) bool {
	return a.Port == o.Port && a.IP.Equal(o.IP)
}

// UDPAddr converts itself to *net.Addr.
func (a CompactAddr) UDPAddr() *net.UDPAddr {
	return &net.UDPAddr{IP: a.IP, Port: int(a.Port)}
}

var _ net.Addr = CompactAddr{}

// Network implements the interface net.Addr#Network.
func (a CompactAddr) Network() string {
	return "krpc"
}

func (a CompactAddr) String() string {
	if a.Port == 0 {
		return a.IP.String()
	}
	return net.JoinHostPort(a.IP.String(), strconv.FormatUint(uint64(a.Port), 10))
}

// WriteBinary is the same as MarshalBinary, but writes the result into w
// instead of returning.
func (a CompactAddr) WriteBinary(w io.Writer) (n int, err error) {
	if n, err = w.Write(a.IP); err == nil {
		if err = binary.Write(w, binary.BigEndian, a.Port); err == nil {
			n += 2
		}
	}
	return
}

var (
	_ encoding.BinaryMarshaler   = new(CompactAddr)
	_ encoding.BinaryUnmarshaler = new(CompactAddr)
)

// MarshalBinary implements the interface encoding.BinaryMarshaler,
func (a CompactAddr) MarshalBinary() (data []byte, err error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(18)
	if _, err = a.WriteBinary(buf); err == nil {
		data = buf.Bytes()
	}
	return
}

// UnmarshalBinary implements the interface encoding.BinaryUnmarshaler.
func (a *CompactAddr) UnmarshalBinary(data []byte) error {
	_len := len(data) - 2
	switch _len {
	case net.IPv4len, net.IPv6len:
	default:
		return errors.New("invalid compact ip-address/port info")
	}

	a.IP = make(net.IP, _len)
	copy(a.IP, data[:_len])
	a.Port = binary.BigEndian.Uint16(data[_len:])
	return nil
}

var (
	_ bencode.Marshaler   = new(CompactAddr)
	_ bencode.Unmarshaler = new(CompactAddr)
)

// MarshalBencode implements the interface bencode.Marshaler.
func (a CompactAddr) MarshalBencode() (b []byte, err error) {
	if b, err = a.MarshalBinary(); err == nil {
		b, err = bencode.EncodeBytes(b)
	}
	return
}

// UnmarshalBencode implements the interface bencode.Unmarshaler.
func (a *CompactAddr) UnmarshalBencode(b []byte) (err error) {
	var data []byte
	if err = bencode.DecodeBytes(b, &data); err == nil {
		err = a.UnmarshalBinary(data)
	}
	return
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

var (
	_ bencode.Marshaler          = new(CompactIPv4Addrs)
	_ bencode.Unmarshaler        = new(CompactIPv4Addrs)
	_ encoding.BinaryMarshaler   = new(CompactIPv4Addrs)
	_ encoding.BinaryUnmarshaler = new(CompactIPv4Addrs)
)

// CompactIPv4Addrs is a set of IPv4 Addrs.
type CompactIPv4Addrs []CompactAddr

// MarshalBinary implements the interface encoding.BinaryMarshaler.
func (cas CompactIPv4Addrs) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(6 * len(cas))
	for _, addr := range cas {
		if addr.IP = addr.IP.To4(); len(addr.IP) == 0 {
			continue
		}

		if n, err := addr.WriteBinary(buf); err != nil {
			return nil, err
		} else if n != 6 {
			panic(fmt.Errorf("CompactIPv4Nodes: the invalid node info length '%d'", n))
		}
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary implements the interface encoding.BinaryUnmarshaler.
func (cas *CompactIPv4Addrs) UnmarshalBinary(b []byte) (err error) {
	_len := len(b)
	if _len%6 != 0 {
		return fmt.Errorf("CompactIPv4Addrs: invalid addr info length '%d'", _len)
	}

	addrs := make(CompactIPv4Addrs, 0, _len/6)
	for i := 0; i < _len; i += 6 {
		var addr CompactAddr
		if err = addr.UnmarshalBinary(b[i : i+6]); err != nil {
			return
		}
		addrs = append(addrs, addr)
	}

	*cas = addrs
	return
}

// MarshalBencode implements the interface bencode.Marshaler.
func (cas CompactIPv4Addrs) MarshalBencode() (b []byte, err error) {
	if b, err = cas.MarshalBinary(); err == nil {
		b, err = bencode.EncodeBytes(b)
	}
	return
}

// UnmarshalBencode implements the interface bencode.Unmarshaler.
func (cas *CompactIPv4Addrs) UnmarshalBencode(b []byte) (err error) {
	var data []byte
	if err = bencode.DecodeBytes(b, &data); err == nil {
		err = cas.UnmarshalBinary(data)
	}
	return
}

// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

var (
	_ bencode.Marshaler          = new(CompactIPv6Addrs)
	_ bencode.Unmarshaler        = new(CompactIPv6Addrs)
	_ encoding.BinaryMarshaler   = new(CompactIPv6Addrs)
	_ encoding.BinaryUnmarshaler = new(CompactIPv6Addrs)
)

// CompactIPv6Addrs is a set of IPv6 Addrs.
type CompactIPv6Addrs []CompactAddr

// MarshalBinary implements the interface encoding.BinaryMarshaler.
func (cas CompactIPv6Addrs) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(18 * len(cas))
	for _, addr := range cas {
		addr.IP = addr.IP.To16()
		if n, err := addr.WriteBinary(buf); err != nil {
			return nil, err
		} else if n != 18 {
			panic(fmt.Errorf("CompactIPv4Nodes: the invalid node info length '%d'", n))
		}
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary implements the interface encoding.BinaryUnmarshaler.
func (cas *CompactIPv6Addrs) UnmarshalBinary(b []byte) (err error) {
	_len := len(b)
	if _len%18 != 0 {
		return fmt.Errorf("CompactIPv4Addrs: invalid addr info length '%d'", _len)
	}

	addrs := make(CompactIPv6Addrs, 0, _len/18)
	for i := 0; i < _len; i += 18 {
		var addr CompactAddr
		if err = addr.UnmarshalBinary(b[i : i+18]); err != nil {
			return
		}
		addrs = append(addrs, addr)
	}

	*cas = addrs
	return
}

// MarshalBencode implements the interface bencode.Marshaler.
func (cas CompactIPv6Addrs) MarshalBencode() (b []byte, err error) {
	if b, err = cas.MarshalBinary(); err == nil {
		b, err = bencode.EncodeBytes(b)
	}
	return
}

// UnmarshalBencode implements the interface bencode.Unmarshaler.
func (cas *CompactIPv6Addrs) UnmarshalBencode(b []byte) (err error) {
	var data []byte
	if err = bencode.DecodeBytes(b, &data); err == nil {
		err = cas.UnmarshalBinary(data)
	}
	return
}
