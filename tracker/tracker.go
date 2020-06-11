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

// Package tracker supplies some common type interfaces of the BT tracker
// protocol.
package tracker

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/url"

	"github.com/xgfone/bt/metainfo"
	"github.com/xgfone/bt/tracker/httptracker"
	"github.com/xgfone/bt/tracker/udptracker"
)

// Predefine some announce events.
//
// BEP 3
const (
	None      uint32 = iota
	Completed        // The local peer just completed the torrent.
	Started          // The local peer has just resumed this torrent.
	Stopped          // The local peer is leaving the swarm.
)

// AnnounceRequest is the common Announce request.
//
// BEP 3, 15
type AnnounceRequest struct {
	InfoHash metainfo.Hash
	PeerID   metainfo.Hash

	Uploaded   int64
	Downloaded int64
	Left       int64
	Event      uint32

	IP      net.IP
	Key     int32
	NumWant int32 // -1 for default
	Port    uint16
}

// ToHTTPAnnounceRequest creates a new httptracker.AnnounceRequest from itself.
func (ar AnnounceRequest) ToHTTPAnnounceRequest() httptracker.AnnounceRequest {
	var ip string
	if len(ar.IP) != 0 {
		ip = ar.IP.String()
	}

	return httptracker.AnnounceRequest{
		InfoHash:   ar.InfoHash,
		PeerID:     ar.PeerID,
		Uploaded:   ar.Uploaded,
		Downloaded: ar.Downloaded,
		Left:       ar.Left,
		Port:       ar.Port,
		IP:         ip,
		Event:      ar.Event,
		NumWant:    ar.NumWant,
		Key:        ar.Key,
	}
}

// ToUDPAnnounceRequest creates a new udptracker.AnnounceRequest from itself.
func (ar AnnounceRequest) ToUDPAnnounceRequest() udptracker.AnnounceRequest {
	return udptracker.AnnounceRequest{
		InfoHash:   ar.InfoHash,
		PeerID:     ar.PeerID,
		Downloaded: ar.Downloaded,
		Left:       ar.Left,
		Uploaded:   ar.Uploaded,
		Event:      ar.Event,
		IP:         ar.IP,
		Key:        ar.Key,
		NumWant:    ar.NumWant,
		Port:       ar.Port,
	}
}

// AnnounceResponse is a common Announce response.
//
// BEP 3, 15
type AnnounceResponse struct {
	Interval  uint32
	Leechers  uint32
	Seeders   uint32
	Addresses []metainfo.Address
}

// FromHTTPAnnounceResponse sets itself from r.
func (ar *AnnounceResponse) FromHTTPAnnounceResponse(r httptracker.AnnounceResponse) {
	ar.Interval = r.Interval
	ar.Leechers = r.Incomplete
	ar.Seeders = r.Complete
	ar.Addresses = make([]metainfo.Address, 0, len(r.Peers)+len(r.Peers6))
	for _, peer := range r.Peers {
		addrs, _ := peer.Addresses()
		ar.Addresses = append(ar.Addresses, addrs...)
	}
	for _, peer := range r.Peers6 {
		addrs, _ := peer.Addresses()
		ar.Addresses = append(ar.Addresses, addrs...)
	}
}

// FromUDPAnnounceResponse sets itself from r.
func (ar *AnnounceResponse) FromUDPAnnounceResponse(r udptracker.AnnounceResponse) {
	ar.Interval = r.Interval
	ar.Leechers = r.Leechers
	ar.Seeders = r.Seeders
	ar.Addresses = r.Addresses
}

// ScrapeResponseResult is a commont Scrape response result.
type ScrapeResponseResult struct {
	// Seeders is the number of active peers that have completed downloading.
	Seeders uint32 `bencode:"complete"` // BEP 15, 48

	// Leechers is the number of active peers that have not completed downloading.
	Leechers uint32 `bencode:"incomplete"` // BEP 15, 48

	// Completed is the total number of peers that have ever completed downloading.
	Completed uint32 `bencode:"downloaded"` // BEP 15, 48
}

// EncodeTo encodes the response to buf.
func (r ScrapeResponseResult) EncodeTo(buf *bytes.Buffer) {
	binary.Write(buf, binary.BigEndian, r.Seeders)
	binary.Write(buf, binary.BigEndian, r.Completed)
	binary.Write(buf, binary.BigEndian, r.Leechers)
}

// DecodeFrom decodes the response from b.
func (r *ScrapeResponseResult) DecodeFrom(b []byte) {
	r.Seeders = binary.BigEndian.Uint32(b[:4])
	r.Completed = binary.BigEndian.Uint32(b[4:8])
	r.Leechers = binary.BigEndian.Uint32(b[8:12])
}

// ScrapeResponse is a commont Scrape response.
type ScrapeResponse map[metainfo.Hash]ScrapeResponseResult

// FromHTTPScrapeResponse sets itself from r.
func (sr ScrapeResponse) FromHTTPScrapeResponse(r httptracker.ScrapeResponse) {
	for k, v := range r.Files {
		sr[k] = ScrapeResponseResult{
			Seeders:   v.Complete,
			Leechers:  v.Incomplete,
			Completed: v.Downloaded,
		}
	}
}

// FromUDPScrapeResponse sets itself from hs and r.
func (sr ScrapeResponse) FromUDPScrapeResponse(hs []metainfo.Hash,
	r []udptracker.ScrapeResponse) {
	klen := len(hs)
	if _len := len(r); _len < klen {
		klen = _len
	}

	for i := 0; i < klen; i++ {
		sr[hs[i]] = ScrapeResponseResult{
			Seeders:   r[i].Seeders,
			Leechers:  r[i].Leechers,
			Completed: r[i].Completed,
		}
	}
}

// Client is the interface of BT tracker client.
type Client interface {
	Announce(AnnounceRequest) (AnnounceResponse, error)
	Scrape([]metainfo.Hash) (ScrapeResponse, error)
}

// NewClient returns a new Client.
func NewClient(connURL string) (c Client, err error) {
	u, err := url.Parse(connURL)
	if err == nil {
		switch u.Scheme {
		case "http", "https":
			c = &tclient{http: httptracker.NewTrackerClient(connURL, "")}
		case "udp", "udp4", "udp6":
			var utc *udptracker.TrackerClient
			utc, err = udptracker.NewTrackerClientByDial(u.Scheme, u.Host)
			if err == nil {
				var e []udptracker.Extension
				if p := u.RequestURI(); p != "" {
					e = []udptracker.Extension{udptracker.NewURLData([]byte(p))}
				}
				c = &tclient{exts: e, udp: utc}
			}
		default:
			err = fmt.Errorf("unknown url scheme '%s'", u.Scheme)
		}
	}
	return
}

type tclient struct {
	http *httptracker.TrackerClient // BEP 3
	udp  *udptracker.TrackerClient  // BEP 15
	exts []udptracker.Extension     // BEP 41
}

func (c *tclient) Announce(req AnnounceRequest) (resp AnnounceResponse, err error) {
	if c.http != nil {
		var r httptracker.AnnounceResponse
		if r, err = c.http.Announce(req.ToHTTPAnnounceRequest()); err != nil {
			return
		} else if r.FailureReason != "" {
			err = errors.New(r.FailureReason)
			return
		}
		resp.FromHTTPAnnounceResponse(r)
		return
	}

	r := req.ToUDPAnnounceRequest()
	r.Exts = c.exts
	rs, err := c.udp.Announce(r)
	if err == nil {
		resp.FromUDPAnnounceResponse(rs)
	}
	return
}

func (c *tclient) Scrape(hs []metainfo.Hash) (resp ScrapeResponse, err error) {
	if c.http != nil {
		var r httptracker.ScrapeResponse
		if r, err = c.http.Scrape(hs); err != nil {
			return
		} else if r.FailureReason != "" {
			err = errors.New(r.FailureReason)
			return
		}
		resp = make(ScrapeResponse, len(r.Files))
		resp.FromHTTPScrapeResponse(r)
		return
	}

	r, err := c.udp.Scrape(hs)
	if err == nil {
		resp = make(ScrapeResponse, len(r))
		resp.FromUDPScrapeResponse(hs, r)
	}
	return
}