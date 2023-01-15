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

package peerprotocol

import (
	"crypto/sha1"
	"encoding/binary"
	"net"

	"github.com/xgfone/bt/metainfo"
)

// GenerateAllowedFastSet generates some allowed fast set of the torrent file.
//
// Argument:
//
//	set: generated piece set, the length of which is the number to be generated.
//	sz: the number of pieces in torrent.
//	ip: the of the remote peer of the connection.
//	infohash: infohash of torrent.
//
// BEP 6
func GenerateAllowedFastSet(set []uint32, sz uint32, ip net.IP, infohash metainfo.Hash) {
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}

	iplen := len(ip)
	x := make([]byte, 20+iplen)
	for i, j := 0, iplen-1; i < j; i++ { // (1) compatible with IPv4/IPv6
		x[i] = ip[i] & 0xff //              (1)
	}
	// x[iplen-1] = 0 // It is equal to 0 primitively.
	copy(x[iplen:], infohash[:]) // (2)

	for cur, k := 0, len(set); cur < k; {
		sum := sha1.Sum(x)                  // (3)
		x = sum[:]                          // (3)
		for i := 0; i < 5 && cur < k; i++ { // (4)
			j := i * 4                               // (5)
			y := binary.BigEndian.Uint32(x[j : j+4]) // (6)
			index := y % sz                          // (7)
			if !uint32Contains(set, index) {         // (8)
				set[cur] = index //                     (9)
				cur++
			}
		}
	}
}

func uint32Contains(ss []uint32, s uint32) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
