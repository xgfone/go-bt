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

package utils

import (
	"net"
	"strconv"
)

func IPAddr(a net.Addr) string {
	switch a := a.(type) {
	case *net.TCPAddr:
		return a.IP.String()
	case *net.UDPAddr:
		return a.IP.String()
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
