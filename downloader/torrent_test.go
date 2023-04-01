// Copyright 2023 xgfone
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

package downloader

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xgfone/go-bt/metainfo"
	"github.com/xgfone/go-bt/peerprotocol"
	pp "github.com/xgfone/go-bt/peerprotocol"
)

type bep10Handler struct {
	peerprotocol.NoopHandler // For implementing peerprotocol.Handler.

	infodata string
}

func (h bep10Handler) OnExtHandShake(c *peerprotocol.PeerConn) error {
	if _, ok := c.ExtendedHandshakeMsg.M[peerprotocol.ExtendedMessageNameMetadata]; !ok {
		return errors.New("missing the extension 'ut_metadata'")
	}

	return c.SendExtHandshakeMsg(peerprotocol.ExtendedHandshakeMsg{
		M:            map[string]uint8{pp.ExtendedMessageNameMetadata: 2},
		MetadataSize: uint64(len(h.infodata)),
	})
}

func (h bep10Handler) OnPayload(c *peerprotocol.PeerConn, extid uint8, extdata []byte) error {
	if extid != 2 {
		return fmt.Errorf("unknown extension id %d", extid)
	}

	var reqmsg peerprotocol.UtMetadataExtendedMsg
	if err := reqmsg.DecodeFromPayload(extdata); err != nil {
		return err
	}

	if reqmsg.MsgType != peerprotocol.UtMetadataExtendedMsgTypeRequest {
		return errors.New("unsupported ut_metadata extension type")
	}

	startIndex := reqmsg.Piece * BlockSize
	endIndex := startIndex + BlockSize
	if totalSize := len(h.infodata); totalSize < endIndex {
		endIndex = totalSize
	}

	respmsg := peerprotocol.UtMetadataExtendedMsg{
		MsgType: peerprotocol.UtMetadataExtendedMsgTypeData,
		Piece:   reqmsg.Piece,
		Data:    []byte(h.infodata[startIndex:endIndex]),
	}

	data, err := respmsg.EncodeToBytes()
	if err != nil {
		return err
	}

	peerextid := c.ExtendedHandshakeMsg.M[peerprotocol.ExtendedMessageNameMetadata]
	return c.SendExtMsg(peerextid, data)
}

func ExampleTorrentDownloader() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := bep10Handler{infodata: "1234567890"}
	infohash := metainfo.NewHashFromString(handler.infodata)

	// Start the torrent server.
	var serverConfig peerprotocol.Config
	serverConfig.ExtBits.Set(peerprotocol.ExtensionBitExtended)
	server, err := peerprotocol.NewServerByListen("tcp", "127.0.0.1:9010", metainfo.NewRandomHash(), handler, &serverConfig)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer server.Close()

	go server.Run()                    // Start the torrent server.
	time.Sleep(time.Millisecond * 100) // Wait that the torrent server finishes to start.

	// Start the torrent downloader.
	downloaderConfig := &TorrentDownloaderConfig{WorkerNum: 3, DialTimeout: time.Second}
	downloader := NewTorrentDownloader(downloaderConfig)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case result := <-downloader.Response():
				fmt.Println(string(result.InfoBytes))
			}
		}
	}()

	// Start to download the torrent.
	downloader.Request("127.0.0.1", 9010, infohash)

	// Wait to finish the test.
	time.Sleep(time.Second)
	cancel()
	time.Sleep(time.Millisecond * 50)

	// Output:
	// 1234567890
}
