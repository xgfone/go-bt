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
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xgfone/bt/metainfo"
)

// NewTrackerClientByDial returns a new TrackerClient by dialing.
func NewTrackerClientByDial(network, address string, c ...TrackerClientConfig) (
	*TrackerClient, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}

	return NewTrackerClient(conn.(*net.UDPConn), c...), nil
}

// NewTrackerClient returns a new TrackerClient.
func NewTrackerClient(conn *net.UDPConn, c ...TrackerClientConfig) *TrackerClient {
	var conf TrackerClientConfig
	conf.set(c...)
	ipv4 := strings.Contains(conn.LocalAddr().String(), ".")
	return &TrackerClient{conn: conn, conf: conf, ipv4: ipv4}
}

// TrackerClientConfig is used to configure the TrackerClient.
type TrackerClientConfig struct {
	// ReadTimeout is used to receive the response.
	ReadTimeout time.Duration // Default: 5s
	MaxBufSize  int           // Default: 2048
}

func (c *TrackerClientConfig) set(conf ...TrackerClientConfig) {
	if len(conf) > 0 {
		*c = conf[0]
	}

	if c.MaxBufSize <= 0 {
		c.MaxBufSize = 2048
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = time.Second * 5
	}
}

// TrackerClient is a tracker client based on UDP.
//
// Notice: the request is synchronized, that's, the last request is not returned,
// the next request must not be sent.
//
// BEP 15
type TrackerClient struct {
	ipv4 bool
	conf TrackerClientConfig
	conn *net.UDPConn
	last time.Time
	cid  uint64
	tid  uint32
}

// Close closes the UDP tracker client.
func (utc *TrackerClient) Close() { utc.conn.Close() }

func (utc *TrackerClient) readResp(b []byte) (int, error) {
	utc.conn.SetReadDeadline(time.Now().Add(utc.conf.ReadTimeout))
	return utc.conn.Read(b)
}

func (utc *TrackerClient) getTranID() uint32 {
	return atomic.AddUint32(&utc.tid, 1)
}

func (utc *TrackerClient) parseError(b []byte) (tid uint32, reason string) {
	tid = binary.BigEndian.Uint32(b[:4])
	reason = string(b[4:])
	return
}

func (utc *TrackerClient) send(b []byte) (err error) {
	n, err := utc.conn.Write(b)
	if err == nil && n < len(b) {
		err = io.ErrShortWrite
	}
	return
}

func (utc *TrackerClient) connect() (err error) {
	tid := utc.getTranID()
	buf := bytes.NewBuffer(make([]byte, 0, 16))
	binary.Write(buf, binary.BigEndian, ProtocolID)
	binary.Write(buf, binary.BigEndian, ActionConnect)
	binary.Write(buf, binary.BigEndian, tid)
	if err = utc.send(buf.Bytes()); err != nil {
		return
	}

	data := make([]byte, 32)
	n, err := utc.readResp(data)
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

func (utc *TrackerClient) getConnectionID() (cid uint64, err error) {
	cid = utc.cid
	if time.Now().Sub(utc.last) > time.Minute {
		if err = utc.connect(); err == nil {
			cid = utc.cid
		}
	}
	return
}

func (utc *TrackerClient) announce(req AnnounceRequest) (r AnnounceResponse, err error) {
	cid, err := utc.getConnectionID()
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
	n, err := utc.readResp(data)
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
//   1. if it does not connect to the UDP tracker server, it will connect to it,
//      then send the ANNOUNCE request.
//   2. If returning an error, you should retry it.
//      See http://www.bittorrent.org/beps/bep_0015.html#time-outs
func (utc *TrackerClient) Announce(r AnnounceRequest) (AnnounceResponse, error) {
	return utc.announce(r)
}

func (utc *TrackerClient) scrape(infohashes []metainfo.Hash) (rs []ScrapeResponse, err error) {
	cid, err := utc.getConnectionID()
	if err != nil {
		return
	}

	tid := utc.getTranID()
	buf := bytes.NewBuffer(make([]byte, 0, 16+len(infohashes)*20))
	binary.Write(buf, binary.BigEndian, cid)
	binary.Write(buf, binary.BigEndian, ActionScrape)
	binary.Write(buf, binary.BigEndian, tid)
	for _, h := range infohashes {
		buf.Write(h[:])
	}
	if err = utc.send(buf.Bytes()); err != nil {
		return
	}

	data := make([]byte, utc.conf.MaxBufSize)
	n, err := utc.readResp(data)
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
//   1. if it does not connect to the UDP tracker server, it will connect to it,
//      then send the ANNOUNCE request.
//   2. If returning an error, you should retry it.
//      See http://www.bittorrent.org/beps/bep_0015.html#time-outs
func (utc *TrackerClient) Scrape(hs []metainfo.Hash) ([]ScrapeResponse, error) {
	return utc.scrape(hs)
}
