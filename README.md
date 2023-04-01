# BT - Another Implementation For Golang [![Build Status](https://github.com/xgfone/go-bt/actions/workflows/go.yml/badge.svg)](https://github.com/xgfone/go-bt/actions/workflows/go.yml) [![GoDoc](https://pkg.go.dev/badge/github.com/xgfone/go-bt)](https://pkg.go.dev/github.com/xgfone/go-bt) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](https://raw.githubusercontent.com/xgfone/go-bt/master/LICENSE)

A pure golang implementation of [BitTorrent](http://bittorrent.org/beps/bep_0000.html) library, which is inspired by [dht](https://github.com/shiyanhui/dht) and [torrent](https://github.com/anacrolix/torrent).

## Install

```shell
$ go get -u github.com/xgfone/go-bt
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

See [godoc](https://pkg.go.dev/github.com/xgfone/go-bt) or [bttools](https://github.com/xgfone/bttools).
