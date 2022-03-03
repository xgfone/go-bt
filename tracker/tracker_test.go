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

package tracker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/xgfone/bt/metainfo"
	"github.com/xgfone/bt/tracker/udptracker"
)

type testHandler struct{}

func (testHandler) OnConnect(raddr net.Addr) (err error) { return }
func (testHandler) OnAnnounce(raddr net.Addr, req udptracker.AnnounceRequest) (
	r udptracker.AnnounceResponse, err error) {
	if req.Port != 80 {
		err = errors.New("port is not 80")
		return
	}

	if len(req.Exts) > 0 {
		for i, ext := range req.Exts {
			switch ext.Type {
			case udptracker.URLData:
				fmt.Printf("Extensions[%d]: URLData(%s)\n", i, string(ext.Data))
			default:
				fmt.Printf("Extensions[%d]: %s\n", i, ext.Type.String())
			}
		}
	}

	r = udptracker.AnnounceResponse{
		Interval:  1,
		Leechers:  2,
		Seeders:   3,
		Addresses: []metainfo.Address{{IP: &net.IPAddr{IP: net.ParseIP("127.0.0.1")}, Port: 8000}},
	}
	return
}
func (testHandler) OnScrap(raddr net.Addr, infohashes []metainfo.Hash) (
	rs []udptracker.ScrapeResponse, err error) {
	rs = make([]udptracker.ScrapeResponse, len(infohashes))
	for i := range infohashes {
		rs[i] = udptracker.ScrapeResponse{
			Seeders:   uint32(i)*10 + 1,
			Leechers:  uint32(i)*10 + 2,
			Completed: uint32(i)*10 + 3,
		}
	}
	return
}

func ExampleClient() {
	// Start the UDP tracker server
	sconn, err := net.ListenPacket("udp4", "127.0.0.1:8000")
	if err != nil {
		log.Fatal(err)
	}
	server := udptracker.NewServer(sconn, testHandler{})
	defer server.Close()
	go server.Run()

	// Wait for the server to be started
	time.Sleep(time.Second)

	// Create a client and dial to the UDP tracker server.
	client, err := NewClient("udp://127.0.0.1:8000/path?a=1&b=2")
	if err != nil {
		log.Fatal(err)
	}

	// Send the ANNOUNCE request to the UDP tracker server,
	// and get the ANNOUNCE response.
	req := AnnounceRequest{IP: &net.IPAddr{IP: net.ParseIP("127.0.0.1")}, Port: 80}
	resp, err := client.Announce(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Interval: %d\n", resp.Interval)
	fmt.Printf("Leechers: %d\n", resp.Leechers)
	fmt.Printf("Seeders: %d\n", resp.Seeders)
	for i, addr := range resp.Addresses {
		fmt.Printf("Address[%d].IP: %s\n", i, addr.IP.String())
		fmt.Printf("Address[%d].Port: %d\n", i, addr.Port)
	}

	// Send the SCRAPE request to the UDP tracker server,
	// and get the SCRAPE respsone.
	h1 := metainfo.Hash{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	h2 := metainfo.Hash{2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}
	rs, err := client.Scrape(context.Background(), []metainfo.Hash{h1, h2})
	if err != nil {
		log.Fatal(err)
	} else if len(rs) != 2 {
		log.Fatalf("%+v", rs)
	}

	for i, r := range rs {
		fmt.Printf("%s.Seeders: %d\n", i.HexString(), r.Seeders)
		fmt.Printf("%s.Leechers: %d\n", i.HexString(), r.Leechers)
		fmt.Printf("%s.Completed: %d\n", i.HexString(), r.Completed)
	}

	// Unordered output:
	// Extensions[0]: URLData(/path?a=1&b=2)
	// Interval: 1
	// Leechers: 2
	// Seeders: 3
	// Address[0].IP: 127.0.0.1
	// Address[0].Port: 8000
	// 0101010101010101010101010101010101010101.Seeders: 1
	// 0101010101010101010101010101010101010101.Leechers: 2
	// 0101010101010101010101010101010101010101.Completed: 3
	// 0202020202020202020202020202020202020202.Seeders: 11
	// 0202020202020202020202020202020202020202.Leechers: 12
	// 0202020202020202020202020202020202020202.Completed: 13
}
