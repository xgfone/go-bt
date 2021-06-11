# BT - Another Implementation For Golang [![Build Status](https://api.travis-ci.com/xgfone/bt.svg?branch=master)](https://travis-ci.com/github/xgfone/bt) [![GoDoc](https://pkg.go.dev/badge/github.com/xgfone/bt)](https://pkg.go.dev/github.com/xgfone/bt) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](https://raw.githubusercontent.com/xgfone/bt/master/LICENSE)

A pure golang implementation of [BitTorrent](http://bittorrent.org/beps/bep_0000.html) library, which is inspired by [dht](https://github.com/shiyanhui/dht) and [torrent](https://github.com/anacrolix/torrent).


## Install
```shell
$ go get -u github.com/xgfone/bt
```


## Features

- Support `Go1.9+`.
- Support IPv4/IPv6.
- Multi-BEPs implementation.
- Pure Go implementation without `CGO`.
- Only library without any denpendencies. For the command tools, see [bttools](https://github.com/xgfone/bttools).


## The Implemented Specifications
- [x] [**BEP 03:** The BitTorrent Protocol Specification](http://bittorrent.org/beps/bep_0003.html)
- [x] [**BEP 05:** DHT Protocol](http://bittorrent.org/beps/bep_0005.html)
- [x] [**BEP 06:** Fast Extension](http://bittorrent.org/beps/bep_0006.html)
- [x] [**BEP 07:** IPv6 Tracker Extension](http://bittorrent.org/beps/bep_0007.html)
- [x] [**BEP 09:** Extension for Peers to Send Metadata Files](http://bittorrent.org/beps/bep_0009.html)
- [x] [**BEP 10:** Extension Protocol](http://bittorrent.org/beps/bep_0010.html)
- [ ] [**BEP 11:** Peer Exchange (PEX)](http://bittorrent.org/beps/bep_0011.html)
- [x] [**BEP 12:** Multitracker Metadata Extension](http://bittorrent.org/beps/bep_0012.html)
- [x] [**BEP 15:** UDP Tracker Protocol for BitTorrent](http://bittorrent.org/beps/bep_0015.html)
- [x] [**BEP 19:** WebSeed - HTTP/FTP Seeding (GetRight style)](http://bittorrent.org/beps/bep_0019.html) (Only `url-list` in metainfo)
- [x] [**BEP 23:** Tracker Returns Compact Peer Lists](http://bittorrent.org/beps/bep_0023.html)
- [x] [**BEP 32:** IPv6 extension for DHT](http://bittorrent.org/beps/bep_0032.html)
- [ ] [**BEP 33:** DHT scrape](http://bittorrent.org/beps/bep_0033.html)
- [x] [**BEP 41:** UDP Tracker Protocol Extensions](http://bittorrent.org/beps/bep_0041.html)
- [x] [**BEP 43:** Read-only DHT Nodes](http://bittorrent.org/beps/bep_0043.html)
- [ ] [**BEP 44:** Storing arbitrary data in the DHT](http://bittorrent.org/beps/bep_0044.html)
- [x] [**BEP 48:** Tracker Protocol Extension: Scrape](http://bittorrent.org/beps/bep_0048.html)


## Example
See [godoc](https://pkg.go.dev/github.com/xgfone/bt) or [bttools](https://github.com/xgfone/bttools).

### Example 1: Download the file from the remote peer

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/xgfone/bt/downloader"
	"github.com/xgfone/bt/metainfo"
	pp "github.com/xgfone/bt/peerprotocol"
	"github.com/xgfone/bt/tracker"
)

var peeraddr string

func init() {
	flag.StringVar(&peeraddr, "peeraddr", "", "The address of the peer storing the file.")
}

func getPeersFromTrackers(id, infohash metainfo.Hash, trackers []string) (peers []string) {
	c, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	resp := tracker.GetPeers(c, id, infohash, trackers)
	for r := range resp {
		for _, addr := range r.Resp.Addresses {
			addrs := addr.String()
			nonexist := true
			for _, peer := range peers {
				if peer == addrs {
					nonexist = false
					break
				}
			}

			if nonexist {
				peers = append(peers, addrs)
			}
		}
	}

	return
}

func main() {
	flag.Parse()

	torrentfile := os.Args[1]
	mi, err := metainfo.LoadFromFile(torrentfile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	id := metainfo.NewRandomHash()
	infohash := mi.InfoHash()
	info, err := mi.Info()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var peers []string
	if peeraddr != "" {
		peers = []string{peeraddr}
	} else {
		// Get the peers from the trackers in the torrent file.
		trackers := mi.Announces().Unique()
		if len(trackers) == 0 {
			fmt.Println("no trackers")
			return
		}

		peers = getPeersFromTrackers(id, infohash, trackers)
		if len(peers) == 0 {
			fmt.Println("no peers")
			return
		}
	}

	// We save the downloaded file to the current directory.
	w := metainfo.NewWriter("", info, 0)
	defer w.Close()

	// We don't request the blocks from the remote peers concurrently,
	// and it is only an example. But you can do it concurrently.
	dm := newDownloadManager(w, info)
	for peerslen := len(peers); peerslen > 0 && !dm.IsFinished(); {
		peerslen--
		peer := peers[peerslen]
		peers = peers[:peerslen]
		downloadFileFromPeer(peer, id, infohash, dm)
	}
}

func downloadFileFromPeer(peer string, id, infohash metainfo.Hash, dm *downloadManager) {
	pc, err := pp.NewPeerConnByDial(peer, id, infohash, time.Second*3)
	if err != nil {
		log.Printf("fail to dial '%s'", peer)
		return
	}
	defer pc.Close()

	dm.doing = false
	pc.Timeout = time.Second * 10
	if err = pc.Handshake(); err != nil {
		log.Printf("fail to handshake with '%s': %s", peer, err)
		return
	}

	info := dm.writer.Info()
	bdh := downloader.NewBlockDownloadHandler(info, dm.OnBlock, dm.RequestBlock)
	if err = bdh.OnHandShake(pc); err != nil {
		log.Printf("handshake error with '%s': %s", peer, err)
		return
	}

	var msg pp.Message
	for !dm.IsFinished() {
		switch msg, err = pc.ReadMsg(); err {
		case nil:
			switch err = pc.HandleMessage(msg, bdh); err {
			case nil, pp.ErrChoked:
			default:
				log.Printf("fail to handle the msg from '%s': %s", peer, err)
				return
			}
		case io.EOF:
			log.Printf("got EOF from '%s'", peer)
			return
		default:
			log.Printf("fail to read the msg from '%s': %s", peer, err)
			return
		}
	}
}

func newDownloadManager(w metainfo.Writer, info metainfo.Info) *downloadManager {
	length := info.Piece(0).Length()
	return &downloadManager{writer: w, plength: length}
}

type downloadManager struct {
	writer  metainfo.Writer
	pindex  uint32
	poffset uint32
	plength int64
	doing   bool
}

func (dm *downloadManager) IsFinished() bool {
	if dm.pindex >= uint32(dm.writer.Info().CountPieces()) {
		return true
	}
	return false
}

func (dm *downloadManager) OnBlock(index, offset uint32, b []byte) (err error) {
	if dm.pindex != index {
		return fmt.Errorf("inconsistent piece: old=%d, new=%d", dm.pindex, index)
	} else if dm.poffset != offset {
		return fmt.Errorf("inconsistent offset for piece '%d': old=%d, new=%d",
			index, dm.poffset, offset)
	}

	dm.doing = false
	n, err := dm.writer.WriteBlock(index, offset, b)
	if err == nil {
		dm.poffset = offset + uint32(n)
		dm.plength -= int64(n)
	}
	return
}

func (dm *downloadManager) RequestBlock(pc *pp.PeerConn) (err error) {
	if dm.doing {
		return
	}

	if dm.plength <= 0 {
		dm.pindex++
		if dm.IsFinished() {
			return
		}

		dm.poffset = 0
		dm.plength = dm.writer.Info().Piece(int(dm.pindex)).Length()
	}

	index := dm.pindex
	begin := dm.poffset
	length := uint32(downloader.BlockSize)
	if length > uint32(dm.plength) {
		length = uint32(dm.plength)
	}

	log.Printf("Request Block from '%s': index=%d, offset=%d, length=%d",
		pc.RemoteAddr().String(), index, begin, length)
	if err = pc.SendRequest(index, begin, length); err == nil {
		dm.doing = true
	}
	return
}
```
