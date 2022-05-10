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

// Package httptracker implements the tracker protocol based on HTTP/HTTPS.
//
// You can use the package to implement a HTTP tracker server to track the
// information that other peers upload or download the file, or to create
// a HTTP tracker client to communicate with the HTTP tracker server.
package httptracker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/xgfone/bt/bencode"
	"github.com/xgfone/bt/metainfo"
)

// AnnounceRequest is the tracker announce requests.
//
// BEP 3
type AnnounceRequest struct {
	// InfoHash is the sha1 hash of the bencoded form of the info value from the metainfo file.
	InfoHash metainfo.Hash `bencode:"info_hash"` // BEP 3

	// PeerID is the id of the downloader.
	//
	// Each downloader generates its own id at random at the start of a new download.
	PeerID metainfo.Hash `bencode:"peer_id"` // BEP 3

	// Uploaded is the total amount uploaded so far, encoded in base ten ascii.
	Uploaded int64 `bencode:"uploaded"` // BEP 3

	// Downloaded is the total amount downloaded so far, encoded in base ten ascii.
	Downloaded int64 `bencode:"downloaded"` // BEP 3

	// Left is the number of bytes this peer still has to download,
	// encoded in base ten ascii.
	//
	// Note that this can't be computed from downloaded and the file length
	// since it might be a resume, and there's a chance that some of the
	// downloaded data failed an integrity check and had to be re-downloaded.
	//
	// If less than 0, math.MaxInt64 will be used for HTTP trackers instead.
	Left int64 `bencode:"left"` // BEP 3

	// Port is the port that this peer is listening on.
	//
	// Common behavior is for a downloader to try to listen on port 6881,
	// and if that port is taken try 6882, then 6883, etc. and give up after 6889.
	Port uint16 `bencode:"port"` // BEP 3

	// IP is the ip or DNS name which this peer is at, which generally used
	// for the origin if it's on the same machine as the tracker.
	//
	// Optional.
	IP string `bencode:"ip,omitempty"` // BEP 3

	// If not present, this is one of the announcements done at regular intervals.
	// An announcement using started is sent when a download first begins,
	// and one using completed is sent when the download is complete.
	// No completed is sent if the file was complete when started.
	// Downloaders send an announcement using stopped when they cease downloading.
	//
	// Optional
	Event uint32 `bencode:"event,omitempty"` // BEP 3

	// Compact indicates whether it hopes the tracker to return the compact
	// peer lists.
	//
	// Optional
	Compact bool `bencode:"compact,omitempty"` // BEP 23

	// NumWant is the number of peers that the client would like to receive
	// from the tracker. This value is permitted to be zero. If omitted,
	// typically defaults to 50 peers.
	//
	// See https://wiki.theory.org/index.php/BitTorrentSpecification
	//
	// Optional.
	NumWant int32 `bencode:"numwant,omitempty"`

	Key int32 `bencode:"key,omitempty"`
}

// ToQuery converts the Request to URL Query.
func (r AnnounceRequest) ToQuery() (vs url.Values) {
	vs = make(url.Values, 9)
	vs.Set("info_hash", r.InfoHash.BytesString())
	vs.Set("peer_id", r.PeerID.BytesString())
	vs.Set("uploaded", strconv.FormatInt(r.Uploaded, 10))
	vs.Set("downloaded", strconv.FormatInt(r.Downloaded, 10))
	vs.Set("left", strconv.FormatInt(r.Left, 10))

	if r.IP != "" {
		vs.Set("ip", r.IP)
	}
	if r.Event > 0 {
		vs.Set("event", strconv.FormatInt(int64(r.Event), 10))
	}
	if r.Port > 0 {
		vs.Set("port", strconv.FormatUint(uint64(r.Port), 10))
	}
	if r.NumWant != 0 {
		vs.Set("numwant", strconv.FormatUint(uint64(r.NumWant), 10))
	}
	if r.Key != 0 {
		vs.Set("key", strconv.FormatInt(int64(r.Key), 10))
	}

	// BEP 23
	if r.Compact {
		vs.Set("compact", "1")
	} else {
		vs.Set("compact", "0")
	}

	return
}

// FromQuery converts URL Query to itself.
func (r *AnnounceRequest) FromQuery(vs url.Values) (err error) {
	if err = r.InfoHash.FromString(vs.Get("info_hash")); err != nil {
		return
	}

	if err = r.PeerID.FromString(vs.Get("peer_id")); err != nil {
		return
	}

	v, err := strconv.ParseInt(vs.Get("uploaded"), 10, 64)
	if err != nil {
		return
	}
	r.Uploaded = v

	v, err = strconv.ParseInt(vs.Get("downloaded"), 10, 64)
	if err != nil {
		return
	}
	r.Downloaded = v

	v, err = strconv.ParseInt(vs.Get("left"), 10, 64)
	if err != nil {
		return
	}
	r.Left = v

	if s := vs.Get("event"); s != "" {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		r.Event = uint32(v)
	}

	if s := vs.Get("port"); s != "" {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		r.Port = uint16(v)
	}

	if s := vs.Get("numwant"); s != "" {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		r.NumWant = int32(v)
	}

	if s := vs.Get("key"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		r.Key = int32(v)
	}

	r.IP = vs.Get("ip")
	switch vs.Get("compact") {
	case "1":
		r.Compact = true
	case "0":
		r.Compact = false
	}

	return
}

// AnnounceResponse is a announce response.
type AnnounceResponse struct {
	FailureReason string `bencode:"failure reason,omitempty"`

	// Interval is the seconds the downloader should wait before next rerequest.
	Interval uint32 `bencode:"interval,omitempty"` // BEP 3

	// Peers is the list of the peers.
	Peers Peers `bencode:"peers,omitempty"` // BEP 3, BEP 23

	// Peers6 is only used for ipv6 in the compact case.
	Peers6 Peers6 `bencode:"peers6,omitempty"` // BEP 7

	// Where's this specified?
	// Mentioned at https://wiki.theory.org/index.php/BitTorrentSpecification.

	// Complete is the number of peers with the entire file.
	Complete uint32 `bencode:"complete,omitempty"`
	// Incomplete is the number of non-seeder peers.
	Incomplete uint32 `bencode:"incomplete,omitempty"`
	// TrackerID is that the client should send back on its next announcements.
	// If absent and a previous announce sent a tracker id,
	// do not discard the old value; keep using it.
	TrackerID string `bencode:"tracker id,omitempty"`
}

// ScrapeResponseResult is the result of the scraped file.
type ScrapeResponseResult struct {
	// Complete is the number of active peers that have completed downloading.
	Complete uint32 `bencode:"complete"` // BEP 48

	// Incomplete is the number of active peers that have not completed downloading.
	Incomplete uint32 `bencode:"incomplete"` // BEP 48

	// The number of peers that have ever completed downloading.
	Downloaded uint32 `bencode:"downloaded"` // BEP 48
}

// ScrapeResponse represents a Scrape response.
//
// BEP 48
type ScrapeResponse struct {
	FailureReason string `bencode:"failure_reason,omitempty"`

	Files map[metainfo.Hash]ScrapeResponseResult `bencode:"files,omitempty"`
}

// DecodeFrom reads the []byte data from r and decodes them to sr by bencode.
//
// r may be the body of the request from the http client.
func (sr *ScrapeResponse) DecodeFrom(r io.Reader) (err error) {
	return bencode.NewDecoder(r).Decode(sr)
}

// EncodeTo encodes the response to []byte by bencode and write the result into w.
//
// w may be http.ResponseWriter.
func (sr ScrapeResponse) EncodeTo(w io.Writer) (err error) {
	return bencode.NewEncoder(w).Encode(sr)
}

// Client represents a tracker client based on HTTP/HTTPS.
type Client struct {
	Client      *http.Client
	ID          metainfo.Hash
	AnnounceURL string
	ScrapeURL   string
}

// NewClient returns a new HTTPClient.
//
// scrapeURL may be empty, which will replace the "announce" in announceURL
// with "scrape" to generate the scrapeURL.
func NewClient(announceURL, scrapeURL string) *Client {
	if scrapeURL == "" {
		scrapeURL = strings.Replace(announceURL, "announce", "scrape", -1)
	}
	id := metainfo.NewRandomHash()
	return &Client{AnnounceURL: announceURL, ScrapeURL: scrapeURL, ID: id}
}

// Close closes the client, which does nothing at present.
func (t *Client) Close() error   { return nil }
func (t *Client) String() string { return t.AnnounceURL }

func (t *Client) send(c context.Context, u string, vs url.Values, r interface{}) (err error) {
	var url string
	if strings.IndexByte(u, '?') < 0 {
		url = fmt.Sprintf("%s?%s", u, vs.Encode())
	} else {
		url = fmt.Sprintf("%s&%s", u, vs.Encode())
	}

	req, err := NewRequestWithContext(c, http.MethodGet, url, nil)
	if err != nil {
		return
	}

	var resp *http.Response
	if t.Client == nil {
		resp, err = http.DefaultClient.Do(req)
	} else {
		resp, err = t.Client.Do(req)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return
	}

	return bencode.NewDecoder(resp.Body).Decode(r)
}

// Announce sends a Announce request to the tracker.
func (t *Client) Announce(c context.Context, req AnnounceRequest) (resp AnnounceResponse, err error) {
	if req.PeerID.IsZero() {
		if t.ID.IsZero() {
			req.PeerID = metainfo.NewRandomHash()
		} else {
			req.PeerID = t.ID
		}
	}

	err = t.send(c, t.AnnounceURL, req.ToQuery(), &resp)
	return
}

// Scrape sends a Scrape request to the tracker.
func (t *Client) Scrape(c context.Context, infohashes []metainfo.Hash) (resp ScrapeResponse, err error) {
	hs := make([]string, len(infohashes))
	for i, h := range infohashes {
		hs[i] = h.BytesString()
	}

	err = t.send(c, t.ScrapeURL, url.Values{"info_hash": hs}, &resp)
	return
}
