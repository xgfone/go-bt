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

package metainfo

import (
	"bytes"
	"fmt"
	"net"
	"strconv"

	"github.com/xgfone/go-bt/bencode"
)

// HostAddr represents an address based on host and port.
type HostAddr struct {
	Host string
	Port uint16
}

// NewHostAddr returns a new host Addr.
func NewHostAddr(host string, port uint16) HostAddr {
	return HostAddr{Host: host, Port: port}
}

// ParseHostAddr parses a string s to Addr.
func ParseHostAddr(s string) (HostAddr, error) {
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return HostAddr{}, err
	}

	_port, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return HostAddr{}, err
	}

	return NewHostAddr(host, uint16(_port)), nil
}

func (a HostAddr) String() string {
	if a.Port == 0 {
		return a.Host
	}
	return net.JoinHostPort(a.Host, strconv.FormatUint(uint64(a.Port), 10))
}

// Equal reports whether a is equal to o.
func (a HostAddr) Equal(o HostAddr) bool {
	return a.Port == o.Port && a.Host == o.Host
}

var (
	_ bencode.Marshaler   = new(HostAddr)
	_ bencode.Unmarshaler = new(HostAddr)
)

// MarshalBencode implements the interface bencode.Marshaler.
func (a HostAddr) MarshalBencode() (b []byte, err error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(64)
	err = bencode.NewEncoder(buf).Encode([]interface{}{a.Host, a.Port})
	if err == nil {
		b = buf.Bytes()
	}
	return
}

// UnmarshalBencode implements the interface bencode.Unmarshaler.
func (a *HostAddr) UnmarshalBencode(b []byte) (err error) {
	var iface interface{}
	if err = bencode.NewDecoder(bytes.NewBuffer(b)).Decode(&iface); err != nil {
		return
	}

	switch v := iface.(type) {
	case string:
		*a, err = ParseHostAddr(v)

	case []interface{}:
		err = a.decode(v)

	default:
		err = fmt.Errorf("unsupported type: %T", iface)
	}

	return
}

func (a *HostAddr) decode(vs []interface{}) (err error) {
	defer func() {
		switch e := recover().(type) {
		case nil:
		case error:
			err = e
		default:
			err = fmt.Errorf("%v", e)
		}
	}()

	a.Host = vs[0].(string)
	a.Port = uint16(vs[1].(int64))
	return
}
