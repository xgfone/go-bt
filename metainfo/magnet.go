// Mozilla Public License Version 2.0
// Modify from github.com/anacrolix/torrent/metainfo.

package metainfo

import (
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Magnet link components.
type Magnet struct {
	InfoHash    Hash       // From "xt"
	Trackers    []string   // From "tr"
	DisplayName string     // From "dn" if not empty
	Params      url.Values // All other values, such as "as", "xs", etc
}

const xtPrefix = "urn:btih:"

// Peers returns the list of the addresses of the peers.
//
// See BEP 9
func (m Magnet) Peers() (peers []HostAddress, err error) {
	vs := m.Params["x.pe"]
	peers = make([]HostAddress, 0, len(vs))
	for _, v := range vs {
		if v != "" {
			var addr HostAddress
			if err = addr.FromString(v); err != nil {
				return
			}

			peers = append(peers, addr)
		}
	}
	return
}

func (m Magnet) String() string {
	vs := make(url.Values, len(m.Params)+len(m.Trackers)+2)
	for k, v := range m.Params {
		vs[k] = append([]string(nil), v...)
	}

	for _, tr := range m.Trackers {
		vs.Add("tr", tr)
	}
	if m.DisplayName != "" {
		vs.Add("dn", m.DisplayName)
	}

	// Transmission and Deluge both expect "urn:btih:" to be unescaped.
	// Deluge wants it to be at the start of the magnet link.
	// The InfoHash field is expected to be BitTorrent in this implementation.
	u := url.URL{
		Scheme:   "magnet",
		RawQuery: "xt=" + xtPrefix + m.InfoHash.HexString(),
	}
	if len(vs) != 0 {
		u.RawQuery += "&" + vs.Encode()
	}
	return u.String()
}

// ParseMagnetURI parses Magnet-formatted URIs into a Magnet instance.
func ParseMagnetURI(uri string) (m Magnet, err error) {
	u, err := url.Parse(uri)
	if err != nil {
		err = fmt.Errorf("error parsing uri: %w", err)
		return
	} else if u.Scheme != "magnet" {
		err = fmt.Errorf("unexpected scheme %q", u.Scheme)
		return
	}

	q := u.Query()
	xt := q.Get("xt")
	if m.InfoHash, err = parseInfohash(q.Get("xt")); err != nil {
		err = fmt.Errorf("error parsing infohash %q: %w", xt, err)
		return
	}
	dropFirst(q, "xt")

	m.DisplayName = q.Get("dn")
	dropFirst(q, "dn")

	m.Trackers = q["tr"]
	delete(q, "tr")

	if len(q) == 0 {
		q = nil
	}

	m.Params = q
	return
}

func parseInfohash(xt string) (ih Hash, err error) {
	if !strings.HasPrefix(xt, xtPrefix) {
		err = errors.New("bad xt parameter prefix")
		return
	}

	var n int
	encoded := xt[len(xtPrefix):]
	switch len(encoded) {
	case 40:
		n, err = hex.Decode(ih[:], []byte(encoded))
	case 32:
		n, err = base32.StdEncoding.Decode(ih[:], []byte(encoded))
	default:
		err = fmt.Errorf("unhandled xt parameter encoding (encoded length %d)", len(encoded))
		return
	}

	if err != nil {
		err = fmt.Errorf("error decoding xt: %w", err)
	} else if n != 20 {
		panic(fmt.Errorf("invalid length '%d' of the decoded bytes", n))
	}

	return
}

func dropFirst(vs url.Values, key string) {
	sl := vs[key]
	switch len(sl) {
	case 0, 1:
		vs.Del(key)
	default:
		vs[key] = sl[1:]
	}
}
