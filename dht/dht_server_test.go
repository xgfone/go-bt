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

package dht

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/xgfone/bt/metainfo"
)

type testPeerManager struct {
	lock  sync.RWMutex
	peers map[metainfo.Hash][]metainfo.Address
}

func newTestPeerManager() *testPeerManager {
	return &testPeerManager{peers: make(map[metainfo.Hash][]metainfo.Address)}
}

func (pm *testPeerManager) AddPeer(infohash metainfo.Hash, addr metainfo.Address) {
	pm.lock.Lock()
	var exist bool
	for _, orig := range pm.peers[infohash] {
		if orig.Equal(addr) {
			exist = true
			break
		}
	}
	if !exist {
		pm.peers[infohash] = append(pm.peers[infohash], addr)
	}
	pm.lock.Unlock()
}

func (pm *testPeerManager) GetPeers(infohash metainfo.Hash, maxnum int,
	ipv6 bool) (addrs []metainfo.Address) {
	// We only supports IPv4, so ignore the ipv6 argument.
	pm.lock.RLock()
	_addrs := pm.peers[infohash]
	if _len := len(_addrs); _len > 0 {
		if _len > maxnum {
			_len = maxnum
		}
		addrs = _addrs[:_len]
	}
	pm.lock.RUnlock()
	return
}

func onSearch(infohash string, ip net.Addr, port uint16) {
	addr := net.JoinHostPort(ip.String(), strconv.FormatUint(uint64(port), 10))
	fmt.Printf("%s is searching %s\n", addr, infohash)
}

func onTorrent(infohash string, ip net.Addr, port uint16) {
	addr := net.JoinHostPort(ip.String(), strconv.FormatUint(uint64(port), 10))
	fmt.Printf("%s has downloaded %s\n", addr, infohash)
}

func newDHTServer(id metainfo.Hash, addr string, pm PeerManager) (s *Server, err error) {
	conn, err := net.ListenPacket("udp", addr)
	if err == nil {
		c := Config{ID: id, PeerManager: pm, OnSearch: onSearch, OnTorrent: onTorrent}
		s = NewServer(conn, c)
	}
	return
}

func ExampleServer() {
	// For test, we predefine some node ids and infohash.
	id1 := metainfo.NewRandomHash()
	id2 := metainfo.NewRandomHash()
	id3 := metainfo.NewRandomHash()
	infohash := metainfo.Hash{1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		11, 12, 13, 14, 15, 16, 17, 18, 19, 20}

	// Create first DHT server
	pm := newTestPeerManager()
	server1, err := newDHTServer(id1, "127.0.0.1:9001", pm)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer server1.Close()

	// Create second DHT server
	server2, err := newDHTServer(id2, "127.0.0.1:9002", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer server2.Close()

	// Create third DHT server
	server3, err := newDHTServer(id3, "127.0.0.1:9003", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer server3.Close()

	// Start the three DHT servers
	go server1.Run()
	go server2.Run()
	go server3.Run()

	// Wait that the DHT servers start.
	time.Sleep(time.Second)

	// Bootstrap the routing table to create a DHT network with the three servers
	server1.Bootstrap([]string{"127.0.0.1:9002", "127.0.0.1:9003"})
	server2.Bootstrap([]string{"127.0.0.1:9001", "127.0.0.1:9003"})
	server3.Bootstrap([]string{"127.0.0.1:9001", "127.0.0.1:9002"})

	// Wait that the DHT servers learn the routing table
	time.Sleep(time.Second)

	fmt.Println("Server1:", server1.Node4Num())
	fmt.Println("Server2:", server2.Node4Num())
	fmt.Println("Server3:", server3.Node4Num())

	server1.GetPeers(infohash, func(r Result) {
		if len(r.Peers) == 0 {
			fmt.Printf("no peers for %s\n", infohash)
		} else {
			for _, peer := range r.Peers {
				fmt.Printf("%s: %s\n", infohash, peer.String())
			}
		}
	})

	// Wait that the last get_peers ends.
	time.Sleep(time.Second * 2)

	// Add the peer to let the DHT server1 has the peer.
	pm.AddPeer(infohash, metainfo.NewAddress(net.ParseIP("127.0.0.1"), 9001))

	// Search the torrent infohash again, but from DHT server2,
	// which will search the DHT server1 recursively.
	server2.GetPeers(infohash, func(r Result) {
		if len(r.Peers) == 0 {
			fmt.Printf("no peers for %s\n", infohash)
		} else {
			for _, peer := range r.Peers {
				fmt.Printf("%s: %s\n", infohash, peer.String())
			}
		}
	})

	// Wait that the recursive call ends.
	time.Sleep(time.Second * 2)

	// Unordered output:
	// Server1: 2
	// Server2: 2
	// Server3: 2
	// 127.0.0.1:9001 is searching 0102030405060708090a0b0c0d0e0f1011121314
	// 127.0.0.1:9001 is searching 0102030405060708090a0b0c0d0e0f1011121314
	// no peers for 0102030405060708090a0b0c0d0e0f1011121314
	// no peers for 0102030405060708090a0b0c0d0e0f1011121314
	// 127.0.0.1:9002 is searching 0102030405060708090a0b0c0d0e0f1011121314
	// 127.0.0.1:9002 is searching 0102030405060708090a0b0c0d0e0f1011121314
	// no peers for 0102030405060708090a0b0c0d0e0f1011121314
	// 0102030405060708090a0b0c0d0e0f1011121314: 127.0.0.1:9001
	// 127.0.0.1:9001 has downloaded 0102030405060708090a0b0c0d0e0f1011121314
}
