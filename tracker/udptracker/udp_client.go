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

package udptracker

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xgfone/bt/metainfo"
)

// NewClientByDial returns a new Client by dialing.
func NewClientByDial(network, address string, id metainfo.Hash) (*Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewClient(conn.(*net.UDPConn), id), nil
}

// NewClient returns a new Client.
func NewClient(conn *net.UDPConn, id metainfo.Hash) *Client {
	ipv4 := strings.Contains(conn.LocalAddr().String(), ".")
	if id.IsZero() {
		id = metainfo.NewRandomHash()
	}

	return &Client{
		MaxBufSize: maxBufSize,

		conn: conn,
		ipv4: ipv4,
		id:   id,
	}
}

// Client is a tracker client based on UDP.
//
// Notice: the request is synchronized, that's, the last request is not returned,
// the next request must not be sent.
//
// BEP 15
type Client struct {
	MaxBufSize int // Default: 2048

	id   metainfo.Hash
	conn *net.UDPConn
	last time.Time
	cid  uint64
	tid  uint32
	ipv4 bool
}

// Close closes the UDP tracker client.
func (utc *Client) Close() error   { return utc.conn.Close() }
func (utc *Client) String() string { return utc.conn.RemoteAddr().String() }

func (utc *Client) readResp(ctx context.Context, b []byte) (int, error) {
	done := make(chan struct{})

	var n int
	var err error
	go func() {
		n, err = utc.conn.Read(b)
		close(done)
	}()

	select {
	case <-ctx.Done():
		utc.conn.SetReadDeadline(time.Now())
		return n, ctx.Err()
	case <-done:
		return n, err
	}
}

func (utc *Client) getTranID() uint32 {
	return atomic.AddUint32(&utc.tid, 1)
}

func (utc *Client) parseError(b []byte) (tid uint32, reason string) {
	tid = binary.BigEndian.Uint32(b[:4])
	reason = string(b[4:])
	return
}

func (utc *Client) send(b []byte) (err error) {
	n, err := utc.conn.Write(b)
	if err == nil && n < len(b) {
		err = io.ErrShortWrite
	}
	return
}

func (utc *Client) connect(ctx context.Context) (err error) {
	tid := utc.getTranID()
	buf := bytes.NewBuffer(make([]byte, 0, 16))
	binary.Write(buf, binary.BigEndian, ProtocolID)
	binary.Write(buf, binary.BigEndian, ActionConnect)
	binary.Write(buf, binary.BigEndian, tid)
	if err = utc.send(buf.Bytes()); err != nil {
		return
	}

	data := make([]byte, 32)
	n, err := utc.readResp(ctx, data)
	if err != nil {
		return
	} else if n < 8 {
		return io.ErrShortBuffer
	}

	data = data[:n]
	switch action := binary.BigEndian.Uint32(data[:4]); action {
	case ActionConnect:
	case ActionError:
		_, reason := utc.parseError(data[4:])
		return errors.New(reason)
	default:
		return fmt.Errorf("tracker response is not connect action: %d", action)
	}

	if n < 16 {
		return io.ErrShortBuffer
	}

	if _tid := binary.BigEndian.Uint32(data[4:8]); _tid != tid {
		return fmt.Errorf("expect transaction id %d, but got %d", tid, _tid)
	}

	utc.cid = binary.BigEndian.Uint64(data[8:16])
	utc.last = time.Now()
	return
}

func (utc *Client) getConnectionID(ctx context.Context) (cid uint64, err error) {
	cid = utc.cid
	if time.Now().Sub(utc.last) > time.Minute {
		if err = utc.connect(ctx); err == nil {
			cid = utc.cid
		}
	}
	return
}

func (utc *Client) announce(ctx context.Context, req AnnounceRequest) (r AnnounceResponse, err error) {
	cid, err := utc.getConnectionID(ctx)
	if err != nil {
		return
	}

	tid := utc.getTranID()
	buf := bytes.NewBuffer(make([]byte, 0, 110))
	binary.Write(buf, binary.BigEndian, cid)            // 8: 0  - 8
	binary.Write(buf, binary.BigEndian, ActionAnnounce) // 4: 8  - 12
	binary.Write(buf, binary.BigEndian, tid)            // 4: 12 - 16
	req.EncodeTo(buf)
	b := buf.Bytes()
	if err = utc.send(b); err != nil {
		return
	}

	bufSize := utc.MaxBufSize
	if bufSize <= 0 {
		bufSize = maxBufSize
	}

	data := make([]byte, bufSize)
	n, err := utc.readResp(ctx, data)
	if err != nil {
		return
	} else if n < 8 {
		err = io.ErrShortBuffer
		return
	}

	data = data[:n]
	switch action := binary.BigEndian.Uint32(data[:4]); action {
	case ActionAnnounce:
	case ActionError:
		_, reason := utc.parseError(data[4:])
		err = errors.New(reason)
		return
	default:
		err = fmt.Errorf("tracker response is not connect action: %d", action)
		return
	}

	if n < 16 {
		err = io.ErrShortBuffer
		return
	}

	if _tid := binary.BigEndian.Uint32(data[4:8]); _tid != tid {
		err = fmt.Errorf("expect transaction id %d, but got %d", tid, _tid)
		return
	}

	r.DecodeFrom(data[8:], utc.ipv4)
	return
}

// Announce sends a Announce request to the tracker.
//
// Notice:
//  1. if it does not connect to the UDP tracker server, it will connect to it,
//     then send the ANNOUNCE request.
//  2. If returning an error, you should retry it.
//     See http://www.bittorrent.org/beps/bep_0015.html#time-outs
func (utc *Client) Announce(c context.Context, r AnnounceRequest) (AnnounceResponse, error) {
	if r.InfoHash.IsZero() {
		panic("infohash is ZERO")
	}

	r.PeerID = utc.id
	return utc.announce(c, r)
}

func (utc *Client) scrape(c context.Context, ihs []metainfo.Hash) (rs []ScrapeResponse, err error) {
	cid, err := utc.getConnectionID(c)
	if err != nil {
		return
	}

	tid := utc.getTranID()
	buf := bytes.NewBuffer(make([]byte, 0, 16+len(ihs)*20))
	binary.Write(buf, binary.BigEndian, cid)          // 8: 0  - 8
	binary.Write(buf, binary.BigEndian, ActionScrape) // 4: 8  - 12
	binary.Write(buf, binary.BigEndian, tid)          // 4: 12 - 16

	for _, h := range ihs { // 20*N: 16 -
		buf.Write(h[:])
	}

	if err = utc.send(buf.Bytes()); err != nil {
		return
	}

	bufSize := utc.MaxBufSize
	if bufSize <= 0 {
		bufSize = maxBufSize
	}

	data := make([]byte, bufSize)
	n, err := utc.readResp(c, data)
	if err != nil {
		return
	} else if n < 8 {
		err = io.ErrShortBuffer
		return
	}

	data = data[:n]
	switch action := binary.BigEndian.Uint32(data[:4]); action {
	case ActionScrape:
	case ActionError:
		_, reason := utc.parseError(data[4:])
		err = errors.New(reason)
		return
	default:
		err = fmt.Errorf("tracker response is not connect action %d", action)
		return
	}

	if _tid := binary.BigEndian.Uint32(data[4:8]); _tid != tid {
		err = fmt.Errorf("expect transaction id %d, but got %d", tid, _tid)
		return
	}

	data = data[8:]
	_len := len(data)
	rs = make([]ScrapeResponse, 0, _len/12)
	for i := 12; i <= _len; i += 12 {
		var r ScrapeResponse
		r.DecodeFrom(data[i-12 : i])
		rs = append(rs, r)
	}

	return
}

// Scrape sends a Scrape request to the tracker.
//
// Notice:
//  1. if it does not connect to the UDP tracker server, it will connect to it,
//     then send the ANNOUNCE request.
//  2. If returning an error, you should retry it.
//     See http://www.bittorrent.org/beps/bep_0015.html#time-outs
func (utc *Client) Scrape(c context.Context, hs []metainfo.Hash) ([]ScrapeResponse, error) {
	return utc.scrape(c, hs)
}
