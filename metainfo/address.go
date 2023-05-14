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

package metainfo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"

	"github.com/eyedeekay/i2pkeys"
	"github.com/eyedeekay/go-i2p-bt/bencode"
	"github.com/eyedeekay/go-i2p-bt/utils"
)

// ErrInvalidAddr is returned when the compact address is invalid.
var ErrInvalidAddr = fmt.Errorf("invalid compact information of ip and port")

// Address represents a client/server listening on a UDP port implementing
// the DHT protocol.
type Address struct {
	IP   net.Addr
	Port uint16
}

// NewAddress returns a new Address.
func NewAddress(ip interface{}, port uint16) Address {
	switch v := ip.(type) {
	case net.IP:
		return Address{IP: &net.IPAddr{
			IP: v,
		}, Port: port}
	case *net.IPAddr:
		return Address{IP: v, Port: port}
	case i2pkeys.I2PAddr:
		return Address{IP: v, Port: port}
	default:
		return Address{IP: nil, Port: 0}
	}
}

// NewAddressFromString returns a new Address by the address string.
func NewAddressFromString(s string) (addr Address, err error) {
	err = addr.FromString(s)
	return
}

func Lookup(shost string, port int) ([]Address, error) {
	if strings.HasSuffix(shost, ".i2p") {
		iaddr, err := i2pkeys.Lookup(shost)
		if err != nil {
			return nil, err
		}
		returnAddresses := []Address{
			{IP: iaddr, Port: 6881},
		}
		return returnAddresses, nil
	}
	ips, err := net.LookupIP(shost)
	if err != nil {
		return nil, fmt.Errorf("fail to lookup the domain '%s': %s", shost, err)
	}
	returnAddrs := make([]Address, len(ips))
	for i, ip := range ips {
		if ipv4 := ip.To4(); len(ipv4) != 0 {
			returnAddrs[i] = Address{
				IP: &net.IPAddr{
					IP: ipv4,
				},
				Port: uint16(port),
			}
		} else {
			returnAddrs[i] = Address{
				IP: &net.IPAddr{
					IP: ip,
				},
				Port: uint16(port),
			}
		}
	}
	return returnAddrs, nil
}

// NewAddressesFromString returns a list of Addresses by the address string.
func NewAddressesFromString(s string) (addrs []Address, err error) {
	shost, sport, err := net.SplitHostPort(s)
	if err != nil {
		if sport == "" && strings.HasSuffix(shost, ".i2p") {
			//Skip the missing port error if the address is an i2p address,
			//since the port is optional and we can assign 6881 automatically.
			sport = "6881"
		} else {
			return nil, fmt.Errorf("invalid address '%s': %s", s, err)
		}
	}

	var port uint16
	if sport != "" {
		v, err := strconv.ParseUint(sport, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid address '%s': %s", s, err)
		}
		port = uint16(v)
	}

	addrs, err = Lookup(shost, int(port))
	if err != nil {
		return nil, fmt.Errorf("fail to lookup the domain '%s': %s", shost, err)
	}
	return
}

func LookupNetAddr(shost string) ([]net.Addr, error) {
	if strings.HasSuffix(shost, ".i2p") {
		iaddr, err := i2pkeys.Lookup(shost)
		if err != nil {
			return nil, err
		}
		returnAddresses := []net.Addr{
			iaddr,
		}
		return returnAddresses, nil
	}
	ips, err := net.LookupIP(shost)
	if err != nil {
		return nil, fmt.Errorf("fail to lookup the domain '%s': %s", shost, err)
	}
	returnAddrs := make([]net.Addr, len(ips))
	for i, ip := range ips {
		if ipv4 := ip.To4(); len(ipv4) != 0 {
			returnAddrs[i] = &net.IPAddr{
				IP: ipv4,
			}
		} else {
			returnAddrs[i] = &net.IPAddr{
				IP: ip,
			}
		}
	}
	return returnAddrs, nil
}

// FromString parses and sets the ip from the string addr.
func (a *Address) FromString(addr string) (err error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		if port == "" && strings.HasSuffix(host, ".i2p") {
			//Skip the missing port error if the address is an i2p address,
			//since the port is optional and we can assign 6881 automatically.
			port = "6881"
		} else {
			return fmt.Errorf("invalid address '%s': %s", addr, err)
		}
	}

	if port != "" {
		v, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			return fmt.Errorf("invalid address '%s': %s", addr, err)
		}
		a.Port = uint16(v)
	}

	ips, err := LookupNetAddr(host)
	if err != nil {
		return fmt.Errorf("fail to lookup the domain '%s': %s", host, err)
	} else if len(ips) == 0 {
		return fmt.Errorf("the domain '%s' has no ips", host)
	}

	a.IP = ips[0]
	return
}

// FromUDPAddr sets the ip from net.UDPAddr.
func (a *Address) FromUDPAddr(ua net.Addr) {
	a.Port = uint16(utils.Port(ua))
	a.IP = ua
}

// Addr creates a new net.Addr.
func (a Address) Addr() net.Addr {
	switch a.IP.(type) {
	case *net.IPAddr:
		return &net.UDPAddr{
			IP:   a.IP.(*net.IPAddr).IP,
			Port: int(a.Port),
		}
	case *net.UDPAddr:
		return &net.UDPAddr{
			IP:   a.IP.(*net.UDPAddr).IP,
			Port: int(a.Port),
		}
	case *i2pkeys.I2PAddr:
		retvalue := a.IP.(*i2pkeys.I2PAddr)
		return retvalue
	case i2pkeys.I2PAddr:
		retvalue := a.IP.(i2pkeys.I2PAddr)
		return &retvalue
	default:
		return nil
	}
}

func (a Address) IsIPv6() bool {
	switch a.IP.(type) {
	case *net.IPAddr:
		return a.IP.(*net.IPAddr).IP.To4() == nil
	case *net.UDPAddr:
		return a.IP.(*net.UDPAddr).IP.To4() == nil
	default:
		return false
	}
}

func (a Address) To4() *net.IPAddr {
	switch a.IP.(type) {
	case *net.IPAddr:
		return &net.IPAddr{
			IP: a.IP.(*net.IPAddr).IP.To4(),
		}
	case *net.UDPAddr:
		return &net.IPAddr{
			IP: a.IP.(*net.UDPAddr).IP.To4(),
		}
	default:
		return &net.IPAddr{
			IP: net.IP{127, 0, 0, 1},
		}
	}
}

func (a Address) To16() *net.IPAddr {
	switch a.IP.(type) {
	case *net.IPAddr:
		return &net.IPAddr{
			IP: a.IP.(*net.IPAddr).IP.To16(),
		}
	case *net.UDPAddr:
		return &net.IPAddr{
			IP: a.IP.(*net.UDPAddr).IP.To16(),
		}
	default:
		return &net.IPAddr{
			IP: net.IPv6loopback,
		}
	}
}

func (a Address) String() string {
	if a.Port == 0 {
		return a.IP.String()
	}
	return net.JoinHostPort(a.IP.String(), strconv.FormatUint(uint64(a.Port), 10))
}

// Equal reports whether n is equal to o, which is equal to
//
//	n.HasIPAndPort(o.IP, o.Port)
func (a Address) Equal(o Address) bool {
	return a.Port == o.Port && a.IP.String() == o.IP.String()
}

// HasIPAndPort reports whether the current node has the ip and the port.
func (a Address) HasIPAndPort(ip net.Addr, port uint16) bool {
	return port == a.Port && a.IP.String() == ip.String()
}

// WriteBinary is the same as MarshalBinary, but writes the result into w
// instead of returning.
func (a Address) WriteBinary(w io.Writer) (m int, err error) {
	var ip []byte
	switch a.IP.(type) {
	case *net.IPAddr:
		ip = []byte(a.IP.(*net.IPAddr).IP)
	case *net.UDPAddr:
		ip = []byte(a.IP.(*net.UDPAddr).IP)
	case i2pkeys.I2PAddr:
		var err error
		ip, err = a.IP.(i2pkeys.I2PAddr).ToBytes()
		if err != nil {
			return 0, err
		}
	}

	if m, err = w.Write(ip); err == nil {
		if err = binary.Write(w, binary.BigEndian, a.Port); err == nil {
			m += 2
		}
	}
	return
}

// UnmarshalBinary implements the interface binary.BinaryUnmarshaler.
func (a *Address) UnmarshalBinary(b []byte) (err error) {
	_len := len(b) - 2
	i2p := false
	if _len <= net.IPv6len {
		switch _len {
		case net.IPv4len, net.IPv6len:
		default:
			return ErrInvalidAddr
		}
	} else {
		_len = len(b)
		i2p = true
	}

	IP := make([]byte, _len)
	copy(IP, b[:_len])
	if i2p {
		a.IP, err = i2pkeys.NewI2PAddrFromBytes(IP)
		if err != nil {
			return err
		}
		a.Port = uint16(6881)
	} else {
		a.IP = &net.IPAddr{
			IP: net.IP(IP),
		}
		a.Port = binary.BigEndian.Uint16(b[_len:])
	}

	return
}

// MarshalBinary implements the interface binary.BinaryMarshaler.
func (a Address) MarshalBinary() (data []byte, err error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(20)
	if _, err = a.WriteBinary(buf); err == nil {
		data = buf.Bytes()
	}
	return
}

func (a *Address) decode(vs []interface{}) (err error) {
	defer func() {
		switch e := recover().(type) {
		case nil:
		case error:
			err = e
		default:
			err = fmt.Errorf("%v", e)
		}
	}()

	host := vs[0].(string)
	if strings.HasSuffix(host, ".i2p") {
		a.IP, err = i2pkeys.NewI2PAddrFromBytes([]byte(host))
		if err != nil {
			return
		}
		a.Port = uint16(6881)
		return
	} else {
		a.IP = &net.IPAddr{
			IP: net.ParseIP(host),
		}
		if len(a.IP.(*net.IPAddr).IP) == 0 {
			return ErrInvalidAddr
		} else if ip := a.IP.(*net.IPAddr).IP.To4(); len(ip) != 0 {
			a.IP = &net.IPAddr{
				IP: ip,
			}
		}
		a.Port = uint16(vs[1].(int64))
		return
	}
}

// UnmarshalBencode implements the interface bencode.Unmarshaler.
func (a *Address) UnmarshalBencode(b []byte) (err error) {
	var iface interface{}
	if err = bencode.NewDecoder(bytes.NewBuffer(b)).Decode(&iface); err != nil {
		return
	}

	switch v := iface.(type) {
	case string:
		err = a.FromString(v)
	case []interface{}:
		err = a.decode(v)
	default:
		err = fmt.Errorf("unsupported type: %T", iface)
	}

	return
}

// MarshalBencode implements the interface bencode.Marshaler.
func (a Address) MarshalBencode() (b []byte, err error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(32)
	err = bencode.NewEncoder(buf).Encode([]interface{}{a.IP.String(), a.Port})
	if err == nil {
		b = buf.Bytes()
	}
	return
}

/// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

// HostAddress is the same as the Address, but the host part may be
// either a domain or a ip.
type HostAddress struct {
	Host string
	Port uint16
}

// NewHostAddress returns a new host addrress.
func NewHostAddress(host string, port uint16) HostAddress {
	return HostAddress{Host: host, Port: port}
}

// NewHostAddressFromString returns a new host address by the string.
func NewHostAddressFromString(s string) (addr HostAddress, err error) {
	err = addr.FromString(s)
	return
}

// FromString parses and sets the host from the string addr.
func (a *HostAddress) FromString(addr string) (err error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address '%s': %s", addr, err)
	} else if host == "" {
		return fmt.Errorf("invalid address '%s': missing host", addr)
	}

	if port != "" {
		v, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			return fmt.Errorf("invalid address '%s': %s", addr, err)
		}
		a.Port = uint16(v)
	}

	a.Host = host
	return
}

func (a HostAddress) String() string {
	if a.Port == 0 {
		return a.Host
	}
	return net.JoinHostPort(a.Host, strconv.FormatUint(uint64(a.Port), 10))
}

// Addresses parses the host address to a list of Addresses.
func (a HostAddress) Addresses() (addrs []Address, err error) {
	if ip := net.ParseIP(a.Host); len(ip) != 0 {
		return []Address{NewAddress(ip, a.Port)}, nil
	}

	ips, err := net.LookupIP(a.Host)
	if err != nil {
		err = fmt.Errorf("fail to lookup the domain '%s': %s", a.Host, err)
	} else {
		addrs = make([]Address, len(ips))
		for i, ip := range ips {
			addrs[i] = NewAddress(ip, a.Port)
		}
	}

	return
}

// Equal reports whether a is equal to o.
func (a HostAddress) Equal(o HostAddress) bool {
	return a.Port == o.Port && a.Host == o.Host
}

func (a *HostAddress) decode(vs []interface{}) (err error) {
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

// UnmarshalBencode implements the interface bencode.Unmarshaler.
func (a *HostAddress) UnmarshalBencode(b []byte) (err error) {
	var iface interface{}
	if err = bencode.NewDecoder(bytes.NewBuffer(b)).Decode(&iface); err != nil {
		return
	}

	switch v := iface.(type) {
	case string:
		err = a.FromString(v)
	case []interface{}:
		err = a.decode(v)
	default:
		err = fmt.Errorf("unsupported type: %T", iface)
	}

	return
}

// MarshalBencode implements the interface bencode.Marshaler.
func (a HostAddress) MarshalBencode() (b []byte, err error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(32)
	err = bencode.NewEncoder(buf).Encode([]interface{}{a.Host, a.Port})
	if err == nil {
		b = buf.Bytes()
	}
	return
}

func ConvertAddress(addr net.Addr) (a Address, err error) {
	switch v := addr.(type) {
	case *net.TCPAddr:
		a = NewAddress(v, uint16(v.Port))
	case *net.UDPAddr:
		a = NewAddress(v, uint16(v.Port))
	case *i2pkeys.I2PAddr:
		a = NewAddress(v, 6881)
	case i2pkeys.I2PAddr:
		a = NewAddress(v, 6881)
	default:
		err = fmt.Errorf("unsupported address type: %T", addr)
	}
	return
}
