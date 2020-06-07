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

package udptracker

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/xgfone/bt/metainfo"
)

type testHandler struct{}

func (testHandler) OnConnect(raddr *net.UDPAddr) (err error) { return }
func (testHandler) OnAnnounce(raddr *net.UDPAddr, req AnnounceRequest) (
	r AnnounceResponse, err error) {
	if req.Port != 80 {
		err = errors.New("port is not 80")
		return
	}

	if len(req.Exts) > 0 {
		for i, ext := range req.Exts {
			switch ext.Type {
			case URLData:
				fmt.Printf("Extensions[%d]: URLData(%s)\n", i, string(ext.Data))
			default:
				fmt.Printf("Extensions[%d]: %s\n", i, ext.Type.String())
			}
		}
	}

	r = AnnounceResponse{
		Interval:  1,
		Leechers:  2,
		Seeders:   3,
		Addresses: []metainfo.Address{{IP: net.ParseIP("127.0.0.1"), Port: 8001}},
	}
	return
}
func (testHandler) OnScrap(raddr *net.UDPAddr, infohashes []metainfo.Hash) (
	rs []ScrapeResponse, err error) {
	rs = make([]ScrapeResponse, len(infohashes))
	for i := range infohashes {
		rs[i] = ScrapeResponse{
			Seeders:   uint32(i)*10 + 1,
			Leechers:  uint32(i)*10 + 2,
			Completed: uint32(i)*10 + 3,
		}
	}
	return
}

func ExampleTrackerClient() {
	// Start the UDP tracker server
	sconn, err := net.ListenPacket("udp4", "127.0.0.1:8001")
	if err != nil {
		log.Fatal(err)
	}
	server := NewTrackerServer(sconn, testHandler{})
	defer server.Close()
	go server.Run()

	// Wait for the server to be started
	time.Sleep(time.Second)

	// Create a client and dial to the UDP tracker server.
	client, err := NewTrackerClientByDial("udp4", "127.0.0.1:8001")
	if err != nil {
		log.Fatal(err)
	}

	// Send the ANNOUNCE request to the UDP tracker server,
	// and get the ANNOUNCE response.
	exts := []Extension{NewURLData([]byte("data")), NewNop()}
	req := AnnounceRequest{IP: net.ParseIP("127.0.0.1"), Port: 80, Exts: exts}
	resp, err := client.Announce(req)
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
	hs := []metainfo.Hash{metainfo.NewRandomHash(), metainfo.NewRandomHash()}
	rs, err := client.Scrape(hs)
	if err != nil {
		log.Fatal(err)
	} else if len(rs) != 2 {
		log.Fatalf("%+v", rs)
	}

	for i, r := range rs {
		fmt.Printf("%d.Seeders: %d\n", i, r.Seeders)
		fmt.Printf("%d.Leechers: %d\n", i, r.Leechers)
		fmt.Printf("%d.Completed: %d\n", i, r.Completed)
	}

	// Output:
	// Extensions[0]: URLData(data)
	// Extensions[1]: Nop
	// Interval: 1
	// Leechers: 2
	// Seeders: 3
	// Address[0].IP: 127.0.0.1
	// Address[0].Port: 8001
	// 0.Seeders: 1
	// 0.Leechers: 2
	// 0.Completed: 3
	// 1.Seeders: 11
	// 1.Leechers: 12
	// 1.Completed: 13
}
