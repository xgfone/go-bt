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

	"github.com/xgfone/bt/metainfo"
	"github.com/xgfone/bt/peerprotocol"
)

const peerBlockSize = 16384 // 16KiB.

// Request is used to send a download request.
type request struct {
	Host     string
	Port     uint16
	PeerID   metainfo.Hash
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

	// WorkerNum is the number of the worker downloading the torrent concurrently.
	//
	// The default is 128.
	WorkerNum int

	// ErrorLog is used to log the error.
	//
	// The default is log.Printf.
	ErrorLog func(format string, args ...interface{})
}

func (c *TorrentDownloaderConfig) set(conf ...TorrentDownloaderConfig) {
	if len(conf) > 0 {
		*c = conf[0]
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
	ebits peerprotocol.ExtensionBits
	ehmsg peerprotocol.ExtendedHandshakeMsg
}

// NewTorrentDownloader returns a new TorrentDownloader.
//
// If id is ZERO, it is reset to a random id. workerNum is 128 by default.
func NewTorrentDownloader(c ...TorrentDownloaderConfig) *TorrentDownloader {
	var conf TorrentDownloaderConfig
	conf.set(c...)

	d := &TorrentDownloader{
		conf:      conf,
		exit:      make(chan struct{}),
		requests:  make(chan request, conf.WorkerNum),
		responses: make(chan TorrentResponse, 1024),

		ehmsg: peerprotocol.ExtendedHandshakeMsg{
			M: map[string]uint8{peerprotocol.ExtendedMessageNameMetadata: 1},
		},
	}

	for i := 0; i < conf.WorkerNum; i++ {
		go d.worker()
	}

	d.ebits.Set(peerprotocol.ExtensionBitExtended)
	return d
}

// Request submits a download request.
//
// Notice: the remote peer must support the "ut_metadata" extenstion.
// Or downloading fails.
func (d *TorrentDownloader) Request(host string, port uint16, infohash metainfo.Hash) {
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
		d.ebits.Unset(peerprotocol.ExtensionBitDHT)
	} else {
		d.ebits.Set(peerprotocol.ExtensionBitDHT)
	}
}

func (d *TorrentDownloader) worker() {
	for {
		select {
		case <-d.exit:
			return
		case r := <-d.requests:
			if err := d.download(r.Host, r.Port, r.PeerID, r.InfoHash); err != nil {
				d.conf.ErrorLog("fail to download the torrent '%s': %s",
					r.InfoHash.HexString(), err)
			}
		}
	}
}

func (d *TorrentDownloader) download(host string, port uint16,
	peerID, infohash metainfo.Hash) (err error) {
	addr := net.JoinHostPort(host, strconv.FormatUint(uint64(port), 10))
	conn, err := peerprotocol.NewPeerConnByDial(d.conf.ID, addr)
	if err != nil {
		return fmt.Errorf("fail to dial to '%s': %s", addr, err)
	}
	defer conn.Close()

	conn.ExtensionBits = d.ebits
	rmsg, err := conn.Handshake(infohash)
	if err != nil || rmsg.InfoHash != infohash || !rmsg.IsSupportExtended() {
		return
	} else if !peerID.IsZero() && peerID != rmsg.PeerID {
		return fmt.Errorf("inconsistent peer id '%s'", rmsg.PeerID.HexString())
	}

	if err = conn.SendExtHandshakeMsg(d.ehmsg); err != nil {
		return
	}

	var pieces [][]byte
	var piecesNum int
	var metadataSize int
	var utmetadataID uint8
	var msg peerprotocol.Message

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
		case peerprotocol.Extended:
		case peerprotocol.Port:
			if d.ondht != nil {
				d.ondht(host, msg.Port)
			}
			continue
		default:
			continue
		}

		switch msg.ExtendedID {
		case peerprotocol.ExtendedIDHandshake:
			if utmetadataID > 0 {
				return fmt.Errorf("rehandshake from the peer '%s'", conn.RemoteAddr().String())
			}

			var ehmsg peerprotocol.ExtendedHandshakeMsg
			if err = ehmsg.Decode(msg.ExtendedPayload); err != nil {
				return
			}

			utmetadataID = ehmsg.M[peerprotocol.ExtendedMessageNameMetadata]
			if utmetadataID == 0 {
				return errors.New(`the peer does not support "ut_metadata"`)
			}

			metadataSize = ehmsg.MetadataSize
			piecesNum = metadataSize / peerBlockSize
			if metadataSize%peerBlockSize != 0 {
				piecesNum++
			}

			pieces = make([][]byte, piecesNum)
			go d.requestPieces(conn, utmetadataID, piecesNum)
		case 1:
			if pieces == nil {
				return
			}

			var utmsg peerprotocol.UtMetadataExtendedMsg
			if err = utmsg.DecodeFromPayload(msg.ExtendedPayload); err != nil {
				return
			}

			if utmsg.MsgType != peerprotocol.UtMetadataExtendedMsgTypeData {
				continue
			}

			pieceLen := len(utmsg.Data)
			if (utmsg.Piece != piecesNum-1 && pieceLen != peerBlockSize) ||
				(utmsg.Piece == piecesNum-1 && pieceLen != metadataSize%peerBlockSize) {
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
						PeerID:    rmsg.PeerID,
						InfoHash:  infohash,
						InfoBytes: metadataInfo,
					}
				}
				return
			}
		}
	}
}

func (d *TorrentDownloader) requestPieces(conn *peerprotocol.PeerConn, utMetadataID uint8, piecesNum int) {
	for i := 0; i < piecesNum; i++ {
		payload, err := peerprotocol.UtMetadataExtendedMsg{
			MsgType: peerprotocol.UtMetadataExtendedMsgTypeRequest,
			Piece:   i,
		}.EncodeToBytes()
		if err != nil {
			panic(err)
		}

		msg := peerprotocol.Message{
			Type:            peerprotocol.Extended,
			ExtendedID:      utMetadataID,
			ExtendedPayload: payload,
		}

		if err = conn.WriteMsg(msg); err != nil {
			d.conf.ErrorLog("fail to send the ut_metadata request: %s", err)
			return
		}
	}
}
