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

package krpc

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/xgfone/go-bt/bencode"
)

// Addr represents an address based on ip and port,
// which implements "Compact IP-address/port info".
//
// See http://bittorrent.org/beps/bep_0005.html.
type Addr struct {
	IP   net.IP // For IPv4, its length must be 4.
	Port uint16

	// The original network address, which is only used by the DHT server.
	Orig net.Addr
}

// ParseAddrs parses the address from the string s with the format "IP:PORT".
func ParseAddrs(s string) (addrs []Addr, err error) {
	_ip, _port, err := net.SplitHostPort(s)
	if err != nil {
		return
	}

	port, err := strconv.ParseUint(_port, 10, 16)
	if err != nil {
		return
	}

	ip := net.ParseIP(_ip)
	if ip != nil {
		if ipv4 := ip.To4(); ipv4 != nil {
			ip = ipv4
		}
		return []Addr{NewAddr(ip, uint16(port))}, nil
	}

	ips, err := net.LookupIP(_ip)
	if err != nil {
		return nil, err
	}

	addrs = make([]Addr, len(ips))
	for i, ip := range ips {
		if ipv4 := ip.To4(); ipv4 != nil {
			ip = ipv4
		}
		addrs[i] = NewAddr(ip, uint16(port))
	}

	return
}

// NewAddr returns a new Addr with ip and port.
func NewAddr(ip net.IP, port uint16) Addr {
	return Addr{IP: ip, Port: port}
}

// NewAddrFromUDPAddr converts *net.UDPAddr to a new Addr.
func NewAddrFromUDPAddr(ua *net.UDPAddr) Addr {
	return Addr{IP: ua.IP, Port: uint16(ua.Port), Orig: ua}
}

// Valid reports whether the addr is valid.
func (a Addr) Valid() bool {
	return len(a.IP) > 0 && a.Port > 0
}

// Equal reports whether a is equal to o.
func (a Addr) Equal(o Addr) bool {
	return a.Port == o.Port && a.IP.Equal(o.IP)
}

// UDPAddr converts itself to *net.UDPAddr.
func (a Addr) UDPAddr() *net.UDPAddr {
	return &net.UDPAddr{IP: a.IP, Port: int(a.Port)}
}

var _ net.Addr = Addr{}

// Network implements the interface net.Addr#Network.
func (a Addr) Network() string {
	return "krpc"
}

func (a Addr) String() string {
	if a.Port == 0 {
		return a.IP.String()
	}
	return net.JoinHostPort(a.IP.String(), strconv.FormatUint(uint64(a.Port), 10))
}

// WriteBinary is the same as MarshalBinary, but writes the result into w
// instead of returning.
func (a Addr) WriteBinary(w io.Writer) (n int, err error) {
	if n, err = w.Write(a.IP); err == nil {
		if err = binary.Write(w, binary.BigEndian, a.Port); err == nil {
			n += 2
		}
	}
	return
}

var (
	_ encoding.BinaryMarshaler   = new(Addr)
	_ encoding.BinaryUnmarshaler = new(Addr)
)

// MarshalBinary implements the interface encoding.BinaryMarshaler,
func (a Addr) MarshalBinary() (data []byte, err error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(18)
	if _, err = a.WriteBinary(buf); err == nil {
		data = buf.Bytes()
	}
	return
}

// UnmarshalBinary implements the interface encoding.BinaryUnmarshaler.
func (a *Addr) UnmarshalBinary(data []byte) error {
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
	_ bencode.Marshaler   = new(Addr)
	_ bencode.Unmarshaler = new(Addr)
)

// MarshalBencode implements the interface bencode.Marshaler.
func (a Addr) MarshalBencode() (b []byte, err error) {
	if b, err = a.MarshalBinary(); err == nil {
		b, err = bencode.EncodeBytes(b)
	}
	return
}

// UnmarshalBencode implements the interface bencode.Unmarshaler.
func (a *Addr) UnmarshalBencode(b []byte) (err error) {
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
type CompactIPv4Addrs []Addr

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
		var addr Addr
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
type CompactIPv6Addrs []Addr

// MarshalBinary implements the interface encoding.BinaryMarshaler.
func (cas CompactIPv6Addrs) MarshalBinary() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(18 * len(cas))
	for _, addr := range cas {
		if addr.IP = addr.IP.To4(); len(addr.IP) == 0 {
			continue
		}

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
		var addr Addr
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
