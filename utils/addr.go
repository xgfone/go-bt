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

package utils

import (
	"net"
	"strconv"

	"github.com/eyedeekay/i2pkeys"
)

func NetIPAddr(a net.Addr) net.IP {
	switch a := a.(type) {
	case *net.TCPAddr:
		return a.IP
	case *net.UDPAddr:
		return a.IP
	case *i2pkeys.I2PAddr:
		return net.ParseIP("127.0.0.1")
	case i2pkeys.I2PAddr:
		return net.ParseIP("127.0.0.1")
	default:
		ip, _, err := net.SplitHostPort(a.String())
		if err != nil {
			return nil
		}
		return net.ParseIP(ip)
	}
}

func IPAddr(a net.Addr) string {
	switch a := a.(type) {
	case *net.TCPAddr:
		return a.IP.String()
	case *net.UDPAddr:
		return a.IP.String()
	case *i2pkeys.I2PAddr:
		return a.DestHash().Hash()
	case i2pkeys.I2PAddr:
		return a.DestHash().Hash()
	default:
		ip, _, err := net.SplitHostPort(a.String())
		if err != nil {
			return ""
		}
		return ip
	}
}

func Port(a net.Addr) int {
	switch a := a.(type) {
	case *net.TCPAddr:
		return a.Port
	case *net.UDPAddr:
		return a.Port
	case *i2pkeys.I2PAddr:
		return 6881
	case i2pkeys.I2PAddr:
		return 6881
	default:
		_, port, err := net.SplitHostPort(a.String())
		if err != nil {
			return 0
		}
		iport, err := strconv.Atoi(port)
		if err != nil {
			return 0
		}
		return iport
	}
}

func IsIPv6Addr(addr net.Addr) bool {
	switch xaddr := addr.(type) {
	case *net.UDPAddr:
		return xaddr.IP.To4() == nil
	case *net.TCPAddr:
		return xaddr.IP.To4() == nil
	case *net.IPAddr:
		return xaddr.IP.To4() == nil
	default:
		return false
	}
}

func IpIsZero(ip net.IP) bool {
	for _, b := range ip {
		if b != 0 {
			return false
		}
	}
	return true
}
