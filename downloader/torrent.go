// Copyright 2020~2023 xgfone
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

// Package downloader is used to download the torrent or the real file
// from the peer node by the peer wire protocol.
package downloader

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/xgfone/bt/metainfo"
	pp "github.com/xgfone/bt/peerprotocol"
)

// BlockSize is the size of a block of the piece.
const BlockSize = 16384 // 16KiB.

// Request is used to send a download request.
type request struct {
	Host string
	Port uint16
	// PeerID   metainfo.Hash
	InfoHash metainfo.Hash
}

// TorrentResponse represents a torrent info response.
type TorrentResponse struct {
	Host      string        // Which host the torrent is downloaded from
	Port      uint16        // Which port the torrent is downloaded from
	PeerID    metainfo.Hash // ID of the peer where torrent is downloaded from
	InfoHash  metainfo.Hash // The SHA-1 hash of the torrent to be downloaded
	InfoBytes []byte        // The content of the info part in the torrent
}

// TorrentDownloaderConfig is used to configure the TorrentDownloader.
type TorrentDownloaderConfig struct {
	// ID is the id of the downloader peer node.
	//
	// The default is a random id.
	ID metainfo.Hash

	// The size of a block of the piece.
	//
	// Default: 16384
	BlockSize uint64

	// WorkerNum is the number of the worker downloading the torrent concurrently.
	//
	// The default is 128.
	WorkerNum int

	// DialTimeout is the timeout used by dialing to the peer on TCP.
	DialTimeout time.Duration

	// ErrorLog is used to log the error.
	//
	// The default is log.Printf.
	ErrorLog func(format string, args ...interface{})
}

func (c *TorrentDownloaderConfig) set(conf *TorrentDownloaderConfig) {
	if conf != nil {
		*c = *conf
	}

	if c.BlockSize <= 0 {
		c.BlockSize = BlockSize
	}
	if c.WorkerNum <= 0 {
		c.WorkerNum = 128
	}
	if c.ID.IsZero() {
		c.ID = metainfo.NewRandomHash()
	}
	if c.ErrorLog == nil {
		c.ErrorLog = log.Printf
	}
}

// TorrentDownloader is used to download the torrent file from the peer.
type TorrentDownloader struct {
	conf      TorrentDownloaderConfig
	exit      chan struct{}
	requests  chan request
	responses chan TorrentResponse

	ondht func(string, uint16)
	ebits pp.ExtensionBits
	ehmsg pp.ExtendedHandshakeMsg
}

// NewTorrentDownloader returns a new TorrentDownloader.
//
// If id is ZERO, it is reset to a random id. workerNum is 128 by default.
func NewTorrentDownloader(c *TorrentDownloaderConfig) *TorrentDownloader {
	var conf TorrentDownloaderConfig
	conf.set(c)

	d := &TorrentDownloader{
		conf:      conf,
		exit:      make(chan struct{}),
		requests:  make(chan request, conf.WorkerNum),
		responses: make(chan TorrentResponse, 1024),
		ehmsg: pp.ExtendedHandshakeMsg{
			M: map[string]uint8{pp.ExtendedMessageNameMetadata: 1},
		},
	}

	for i := 0; i < conf.WorkerNum; i++ {
		go d.worker()
	}

	d.ebits.Set(pp.ExtensionBitExtended)
	return d
}

// Request submits a download request.
//
// Notice: the remote peer must support the "ut_metadata" extenstion.
// Or downloading fails.
func (d *TorrentDownloader) Request(host string, port uint16, infohash metainfo.Hash) {
	if infohash.IsZero() {
		panic("infohash is ZERO")
	}
	d.requests <- request{Host: host, Port: port, InfoHash: infohash}
}

// Response returns a response channel to get the downloaded torrent info.
func (d *TorrentDownloader) Response() <-chan TorrentResponse {
	return d.responses
}

// Close closes the downloader and releases the underlying resources.
func (d *TorrentDownloader) Close() {
	select {
	case <-d.exit:
	default:
		close(d.exit)
	}
}

// OnDHTNode sets the DHT node callback, which will enable DHT extenstion bit.
//
// In the callback function, you maybe ping it like DHT by UDP.
// If the node responds, you can add the node in DHT routing table.
//
// BEP 5
func (d *TorrentDownloader) OnDHTNode(cb func(host string, port uint16)) {
	d.ondht = cb
	if cb == nil {
		d.ebits.Unset(pp.ExtensionBitDHT)
	} else {
		d.ebits.Set(pp.ExtensionBitDHT)
	}
}

func (d *TorrentDownloader) worker() {
	for {
		select {
		case <-d.exit:
			return
		case r := <-d.requests:
			if err := d.download(r.Host, r.Port, r.InfoHash); err != nil {
				d.conf.ErrorLog("fail to download the torrent '%s': %s",
					r.InfoHash.HexString(), err)
			}
		}
	}
}

func (d *TorrentDownloader) download(host string, port uint16, infohash metainfo.Hash) (err error) {
	addr := net.JoinHostPort(host, strconv.FormatUint(uint64(port), 10))
	conn, err := pp.NewPeerConnByDial(addr, d.conf.ID, infohash, d.conf.DialTimeout)
	if err != nil {
		return fmt.Errorf("fail to dial to '%s': %s", addr, err)
	}
	defer conn.Close()

	conn.ExtBits = d.ebits
	if err = conn.Handshake(); err != nil {
		return
	} else if !conn.PeerExtBits.IsSupportExtended() {
		return fmt.Errorf("the remote peer '%s' does not support Extended", addr)
	}

	if err = conn.SendExtHandshakeMsg(d.ehmsg); err != nil {
		return
	}

	var pieces [][]byte
	var piecesNum int
	var metadataSize uint64
	var utmetadataID uint8
	var msg pp.Message

	for {
		if msg, err = conn.ReadMsg(); err != nil {
			return err
		}

		select {
		case <-d.exit:
			return
		default:
		}

		if msg.Keepalive {
			continue
		}

		switch msg.Type {
		case pp.MTypeExtended:
		case pp.MTypePort:
			if d.ondht != nil {
				d.ondht(host, msg.Port)
			}
			continue
		default:
			continue
		}

		switch msg.ExtendedID {
		case pp.ExtendedIDHandshake:
			if utmetadataID > 0 {
				return fmt.Errorf("rehandshake from the peer '%s'", conn.RemoteAddr().String())
			}

			var ehmsg pp.ExtendedHandshakeMsg
			if err = ehmsg.Decode(msg.ExtendedPayload); err != nil {
				return
			}

			utmetadataID = ehmsg.M[pp.ExtendedMessageNameMetadata]
			if utmetadataID == 0 {
				return errors.New(`the peer does not support "ut_metadata"`)
			}

			metadataSize = ehmsg.MetadataSize
			piecesNum = int(metadataSize / d.conf.BlockSize)
			if metadataSize%d.conf.BlockSize != 0 {
				piecesNum++
			}

			pieces = make([][]byte, piecesNum)
			go d.requestPieces(conn, utmetadataID, piecesNum)

		case 1:
			if pieces == nil {
				return
			}

			var utmsg pp.UtMetadataExtendedMsg
			if err = utmsg.DecodeFromPayload(msg.ExtendedPayload); err != nil {
				return
			}

			if utmsg.MsgType != pp.UtMetadataExtendedMsgTypeData {
				continue
			}

			pieceLen := len(utmsg.Data)
			if (utmsg.Piece != piecesNum-1 && pieceLen != int(d.conf.BlockSize)) ||
				(utmsg.Piece == piecesNum-1 && pieceLen != int(metadataSize%d.conf.BlockSize)) {
				return
			}
			pieces[utmsg.Piece] = utmsg.Data

			finish := true
			for _, piece := range pieces {
				if len(piece) == 0 {
					finish = false
					break
				}
			}

			if finish {
				metadataInfo := bytes.Join(pieces, nil)
				if infohash == metainfo.Hash(sha1.Sum(metadataInfo)) {
					d.responses <- TorrentResponse{
						Host:      host,
						Port:      port,
						PeerID:    conn.PeerID,
						InfoHash:  infohash,
						InfoBytes: metadataInfo,
					}
				}
				return
			}
		}
	}
}

func (d *TorrentDownloader) requestPieces(conn *pp.PeerConn, utMetadataID uint8, piecesNum int) {
	for i := 0; i < piecesNum; i++ {
		payload, err := pp.UtMetadataExtendedMsg{
			MsgType: pp.UtMetadataExtendedMsgTypeRequest,
			Piece:   i,
		}.EncodeToBytes()
		if err != nil {
			panic(err)
		}

		msg := pp.Message{
			Type:            pp.MTypeExtended,
			ExtendedID:      utMetadataID,
			ExtendedPayload: payload,
		}

		if err = conn.WriteMsg(msg); err != nil {
			d.conf.ErrorLog("fail to send the ut_metadata request: %s", err)
			return
		}
	}
}
