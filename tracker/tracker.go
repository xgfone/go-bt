// Copyright 2020 xgfone, 2023 idk
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
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/eyedeekay/go-i2p-bt/metainfo"
	"github.com/eyedeekay/go-i2p-bt/tracker/httptracker"
	"github.com/eyedeekay/go-i2p-bt/tracker/udptracker"
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
	InfoHash metainfo.Hash // Required
	PeerID   metainfo.Hash // Required

	Uploaded   int64  // Required, but default: 0, which should be only used for test or first.
	Downloaded int64  // Required, but default: 0, which should be only used for test or first.
	Left       int64  // Required, but default: 0, which should be only used for test or last.
	Event      uint32 // Required, but default: 0

	IP      net.Addr // Optional
	Key     int32    // Optional
	NumWant int32    // Optional, BEP 15: -1 for default. But we use 0 as default.
	Port    uint16   // Optional
}

// ToHTTPAnnounceRequest creates a new httptracker.AnnounceRequest from itself.
func (ar *AnnounceRequest) ToHTTPAnnounceRequest() *httptracker.AnnounceRequest {
	ip := "127.0.0.1"
	if PeerAddress != nil {
		ip = PeerAddress.String()
	} else if ar.IP != nil {
		ip = ar.IP.String()
	}
	if ar.Port == 0 {
		ar.Port = 6881
	}
	return &httptracker.AnnounceRequest{
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
func (ar *AnnounceRequest) ToUDPAnnounceRequest() *udptracker.AnnounceRequest {
	if PeerAddress != nil {
		ar.IP = PeerAddress
	}
	return &udptracker.AnnounceRequest{
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
	Announce(context.Context, AnnounceRequest) (AnnounceResponse, error)
	Scrape(context.Context, []metainfo.Hash) (ScrapeResponse, error)
	String() string
	Close() error
}

// ClientConfig is used to configure the defalut client implementation.
type ClientConfig struct {
	// The ID of the local client peer.
	ID metainfo.Hash

	// The http client used only the tracker client is based on HTTP.
	HTTPClient *http.Client
}

// NewClient returns a new Client.
func NewClient(connURL string, conf ...ClientConfig) (c Client, err error) {
	var config ClientConfig
	if len(conf) > 0 {
		config = conf[0]
	}

	u, err := url.Parse(connURL)
	if err == nil {
		switch u.Scheme {
		case "http", "https":
			tracker := httptracker.NewClient(connURL, "")
			if !config.ID.IsZero() {
				tracker.ID = config.ID
			}
			c = &tclient{url: connURL, http: tracker}

		case "udp", "udp4", "udp6":
			var utc *udptracker.Client
			config := udptracker.ClientConfig{ID: config.ID}
			utc, err = udptracker.NewClientByDial(u.Scheme, u.Host, config)
			if err == nil {
				var e []udptracker.Extension
				if p := u.RequestURI(); p != "" {
					e = []udptracker.Extension{udptracker.NewURLData([]byte(p))}
				}
				c = &tclient{url: connURL, exts: e, udp: utc}
			}
		default:
			err = fmt.Errorf("unknown url scheme '%s'", u.Scheme)
		}
	}
	return
}

type tclient struct {
	url  string
	http *httptracker.Client    // BEP 3
	udp  *udptracker.Client     // BEP 15
	exts []udptracker.Extension // BEP 41
}

func (c *tclient) String() string { return c.url }

func (c *tclient) Close() error {
	if c.http != nil {
		return c.http.Close()
	}
	return c.udp.Close()
}

func (c *tclient) Announce(ctx context.Context, req AnnounceRequest) (resp AnnounceResponse, err error) {
	if c.http != nil {
		var r httptracker.AnnounceResponse
		if r, err = c.http.Announce(ctx, req.ToHTTPAnnounceRequest()); err != nil {
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
	rs, err := c.udp.Announce(ctx, r)
	if err == nil {
		resp.FromUDPAnnounceResponse(rs)
	}
	return
}

func (c *tclient) Scrape(ctx context.Context, hs []metainfo.Hash) (resp ScrapeResponse, err error) {
	if c.http != nil {
		var r httptracker.ScrapeResponse
		if r, err = c.http.Scrape(ctx, hs); err != nil {
			return
		} else if r.FailureReason != "" {
			err = errors.New(r.FailureReason)
			return
		}
		resp = make(ScrapeResponse, len(r.Files))
		resp.FromHTTPScrapeResponse(r)
		return
	}

	r, err := c.udp.Scrape(ctx, hs)
	if err == nil {
		resp = make(ScrapeResponse, len(r))
		resp.FromUDPScrapeResponse(hs, r)
	}
	return
}
