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
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xgfone/bt/metainfo"
)

var Dial = net.Dial

// NewClientByDial returns a new Client by dialing.
func NewClientByDial(network, address string, c ...ClientConfig) (*Client, error) {
	conn, err := Dial(network, address)
	if err != nil {
		return nil, err
	}

	return NewClient(conn, c...), nil
}

// NewClient returns a new Client.
func NewClient(conn net.Conn, c ...ClientConfig) *Client {
	var conf ClientConfig
	conf.set(c...)
	ipv4 := strings.Contains(conn.LocalAddr().String(), ".")
	return &Client{conn: conn, conf: conf, ipv4: ipv4}
}

// ClientConfig is used to configure the Client.
type ClientConfig struct {
	ID         metainfo.Hash
	MaxBufSize int // Default: 2048
}

func (c *ClientConfig) set(conf ...ClientConfig) {
	if len(conf) > 0 {
		*c = conf[0]
	}

	if c.ID.IsZero() {
		c.ID = metainfo.NewRandomHash()
	}
	if c.MaxBufSize <= 0 {
		c.MaxBufSize = 2048
	}
}

// Client is a tracker client based on UDP.
//
// Notice: the request is synchronized, that's, the last request is not returned,
// the next request must not be sent.
//
// BEP 15
type Client struct {
	ipv4 bool
	conf ClientConfig
	conn net.Conn
	last time.Time
	cid  uint64
	tid  uint32
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
	//n, err := utc.conn.WriteTo()
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
	switch binary.BigEndian.Uint32(data[:4]) {
	case ActionConnect:
	case ActionError:
		_, reason := utc.parseError(data[4:])
		return errors.New(reason)
	default:
		return errors.New("tracker response not connect action")
	}

	if n < 16 {
		return io.ErrShortBuffer
	}

	if binary.BigEndian.Uint32(data[4:8]) != tid {
		return errors.New("invalid transaction id")
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

func (utc *Client) announce(ctx context.Context, req AnnounceRequest) (
	r AnnounceResponse, err error) {
	cid, err := utc.getConnectionID(ctx)
	if err != nil {
		return
	}

	tid := utc.getTranID()
	buf := bytes.NewBuffer(make([]byte, 0, 110))
	binary.Write(buf, binary.BigEndian, cid)
	binary.Write(buf, binary.BigEndian, ActionAnnounce)
	binary.Write(buf, binary.BigEndian, tid)
	req.EncodeTo(buf)
	b := buf.Bytes()
	if err = utc.send(b); err != nil {
		return
	}

	data := make([]byte, utc.conf.MaxBufSize)
	n, err := utc.readResp(ctx, data)
	if err != nil {
		return
	} else if n < 8 {
		err = io.ErrShortBuffer
		return
	}

	data = data[:n]
	switch binary.BigEndian.Uint32(data[:4]) {
	case ActionAnnounce:
	case ActionError:
		_, reason := utc.parseError(data[4:])
		err = errors.New(reason)
		return
	default:
		err = errors.New("tracker response not connect action")
		return
	}

	if n < 16 {
		err = io.ErrShortBuffer
		return
	}

	if binary.BigEndian.Uint32(data[4:8]) != tid {
		err = errors.New("invalid transaction id")
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
func (utc *Client) Announce(c context.Context, r *AnnounceRequest) (AnnounceResponse, error) {
	if r.PeerID.IsZero() {
		r.PeerID = utc.conf.ID
	}
	return utc.announce(c, *r)
}

func (utc *Client) scrape(c context.Context, ihs []metainfo.Hash) (
	rs []ScrapeResponse, err error) {
	cid, err := utc.getConnectionID(c)
	if err != nil {
		return
	}

	tid := utc.getTranID()
	buf := bytes.NewBuffer(make([]byte, 0, 16+len(ihs)*20))
	binary.Write(buf, binary.BigEndian, cid)
	binary.Write(buf, binary.BigEndian, ActionScrape)
	binary.Write(buf, binary.BigEndian, tid)
	for _, h := range ihs {
		buf.Write(h[:])
	}
	if err = utc.send(buf.Bytes()); err != nil {
		return
	}

	data := make([]byte, utc.conf.MaxBufSize)
	n, err := utc.readResp(c, data)
	if err != nil {
		return
	} else if n < 8 {
		err = io.ErrShortBuffer
		return
	}

	data = data[:n]
	switch binary.BigEndian.Uint32(data[:4]) {
	case ActionScrape:
	case ActionError:
		_, reason := utc.parseError(data[4:])
		err = errors.New(reason)
		return
	default:
		err = errors.New("tracker response not connect action")
		return
	}

	if binary.BigEndian.Uint32(data[4:8]) != tid {
		err = errors.New("invalid transaction id")
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
