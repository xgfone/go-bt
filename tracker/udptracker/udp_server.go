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
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xgfone/bt/metainfo"
)

// TrackerServerHandler is used to handle the request from the client.
type TrackerServerHandler interface {
	// OnConnect is used to check whether to make the connection or not.
	OnConnect(raddr *net.UDPAddr) (err error)
	OnAnnounce(raddr *net.UDPAddr, req AnnounceRequest) (AnnounceResponse, error)
	OnScrap(raddr *net.UDPAddr, infohashes []metainfo.Hash) ([]ScrapeResponse, error)
}

func encodeResponseHeader(buf *bytes.Buffer, action, tid uint32) {
	binary.Write(buf, binary.BigEndian, action)
	binary.Write(buf, binary.BigEndian, tid)
}

type wrappedPeerAddr struct {
	Addr *net.UDPAddr
	Time time.Time
}

// TrackerServerConfig is used to configure the TrackerServer.
type TrackerServerConfig struct {
	MaxBufSize int                                      // Default: 2048
	ErrorLog   func(format string, args ...interface{}) // Default: log.Printf
}

func (c *TrackerServerConfig) setDefault() {
	if c.MaxBufSize <= 0 {
		c.MaxBufSize = 2048
	}
	if c.ErrorLog == nil {
		c.ErrorLog = log.Printf
	}
}

// TrackerServer is a tracker server based on UDP.
type TrackerServer struct {
	conn    net.PacketConn
	conf    TrackerServerConfig
	handler TrackerServerHandler
	bufpool sync.Pool

	cid   uint64
	exit  chan struct{}
	lock  sync.RWMutex
	conns map[uint64]wrappedPeerAddr
}

// NewTrackerServer returns a new TrackerServer.
func NewTrackerServer(c net.PacketConn, h TrackerServerHandler,
	config ...TrackerServerConfig) *TrackerServer {
	var conf TrackerServerConfig
	if len(config) > 0 {
		conf = config[0]
	}
	conf.setDefault()

	s := &TrackerServer{
		conf:    conf,
		conn:    c,
		handler: h,
		exit:    make(chan struct{}),
		conns:   make(map[uint64]wrappedPeerAddr, 128),
	}
	s.bufpool.New = func() interface{} { return make([]byte, conf.MaxBufSize) }

	return s
}

// Close closes the tracker server.
func (uts *TrackerServer) Close() {
	select {
	case <-uts.exit:
	default:
		close(uts.exit)
		uts.conn.Close()
	}
}

func (uts *TrackerServer) cleanConnectionID(interval time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-uts.exit:
			return
		case now := <-tick.C:
			uts.lock.RLock()
			for cid, wa := range uts.conns {
				if now.Sub(wa.Time) > interval {
					delete(uts.conns, cid)
				}
			}
			uts.lock.RUnlock()
		}
	}
}

// Run starts the tracker server.
func (uts *TrackerServer) Run() {
	go uts.cleanConnectionID(time.Minute * 2)
	for {
		buf := uts.bufpool.Get().([]byte)
		n, raddr, err := uts.conn.ReadFrom(buf)
		if err != nil {
			if !strings.Contains(err.Error(), "closed") {
				uts.conf.ErrorLog("failed to read udp tracker request: %s", err)
			}
			return
		} else if n < 16 {
			continue
		}
		go uts.handleRequest(raddr.(*net.UDPAddr), buf, n)
	}
}

func (uts *TrackerServer) handleRequest(raddr *net.UDPAddr, buf []byte, n int) {
	defer uts.bufpool.Put(buf)
	uts.handlePacket(raddr, buf[:n])
}

func (uts *TrackerServer) send(raddr *net.UDPAddr, b []byte) {
	n, err := uts.conn.WriteTo(b, raddr)
	if err != nil {
		uts.conf.ErrorLog("fail to send the udp tracker response to '%s': %s",
			raddr.String(), err)
	} else if n < len(b) {
		uts.conf.ErrorLog("too short udp tracker response sent to '%s'", raddr.String())
	}
}

func (uts *TrackerServer) getConnectionID() uint64 {
	return atomic.AddUint64(&uts.cid, 1)
}

func (uts *TrackerServer) addConnection(cid uint64, raddr *net.UDPAddr) {
	now := time.Now()
	uts.lock.Lock()
	uts.conns[cid] = wrappedPeerAddr{Addr: raddr, Time: now}
	uts.lock.Unlock()
}

func (uts *TrackerServer) checkConnection(cid uint64, raddr *net.UDPAddr) (ok bool) {
	uts.lock.RLock()
	if w, _ok := uts.conns[cid]; _ok && w.Addr.Port == raddr.Port &&
		bytes.Equal(w.Addr.IP, raddr.IP) {
		ok = true
	}
	uts.lock.RUnlock()
	return
}

func (uts *TrackerServer) sendError(raddr *net.UDPAddr, tid uint32, reason string) {
	buf := bytes.NewBuffer(make([]byte, 0, 8+len(reason)))
	encodeResponseHeader(buf, ActionError, tid)
	buf.WriteString(reason)
	uts.send(raddr, buf.Bytes())
}

func (uts *TrackerServer) sendConnResp(raddr *net.UDPAddr, tid uint32, cid uint64) {
	buf := bytes.NewBuffer(make([]byte, 0, 16))
	encodeResponseHeader(buf, ActionConnect, tid)
	binary.Write(buf, binary.BigEndian, cid)
	uts.send(raddr, buf.Bytes())
}

func (uts *TrackerServer) sendAnnounceResp(raddr *net.UDPAddr, tid uint32,
	resp AnnounceResponse) {
	buf := bytes.NewBuffer(make([]byte, 0, 8+12+len(resp.Addresses)*18))
	encodeResponseHeader(buf, ActionAnnounce, tid)
	resp.EncodeTo(buf)
	uts.send(raddr, buf.Bytes())
}

func (uts *TrackerServer) sendScrapResp(raddr *net.UDPAddr, tid uint32,
	rs []ScrapeResponse) {
	buf := bytes.NewBuffer(make([]byte, 0, 8+len(rs)*12))
	encodeResponseHeader(buf, ActionScrape, tid)
	for _, r := range rs {
		r.EncodeTo(buf)
	}
	uts.send(raddr, buf.Bytes())
}

func (uts *TrackerServer) handlePacket(raddr *net.UDPAddr, b []byte) {
	cid := binary.BigEndian.Uint64(b[:8])
	action := binary.BigEndian.Uint32(b[8:12])
	tid := binary.BigEndian.Uint32(b[12:16])
	b = b[16:]

	// Handle the connection request.
	if cid == ProtocolID && action == ActionConnect {
		if err := uts.handler.OnConnect(raddr); err != nil {
			uts.sendError(raddr, tid, err.Error())
			return
		}

		cid := uts.getConnectionID()
		uts.addConnection(cid, raddr)
		uts.sendConnResp(raddr, tid, cid)
		return
	}

	// Check whether the request is connected.
	if !uts.checkConnection(cid, raddr) {
		uts.sendError(raddr, tid, "connection is expired")
		return
	}

	switch action {
	case ActionAnnounce:
		var req AnnounceRequest
		if raddr.IP.To4() != nil { // For ipv4
			if len(b) < 82 {
				uts.sendError(raddr, tid, "invalid announce request")
				return
			}
			req.DecodeFrom(b, true)
		} else { // For ipv6
			if len(b) < 94 {
				uts.sendError(raddr, tid, "invalid announce request")
				return
			}
			req.DecodeFrom(b, false)
		}

		resp, err := uts.handler.OnAnnounce(raddr, req)
		if err != nil {
			uts.sendError(raddr, tid, err.Error())
		} else {
			uts.sendAnnounceResp(raddr, tid, resp)
		}
	case ActionScrape:
		_len := len(b)
		infohashes := make([]metainfo.Hash, 0, _len/20)
		for i, _len := 20, len(b); i <= _len; i += 20 {
			infohashes = append(infohashes, metainfo.NewHash(b[i-20:i]))
		}

		if len(infohashes) == 0 {
			uts.sendError(raddr, tid, "no infohash")
			return
		}

		resps, err := uts.handler.OnScrap(raddr, infohashes)
		if err != nil {
			uts.sendError(raddr, tid, err.Error())
		} else {
			uts.sendScrapResp(raddr, tid, resps)
		}
	default:
		uts.sendError(raddr, tid, "unkwnown action")
	}
}
