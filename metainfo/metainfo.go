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

package metainfo

import (
	"errors"
	"io"
	"os"
	"strings"

	"github.com/xgfone/bt/bencode"
	"github.com/xgfone/bt/internal/helper"
)

// Bytes is the []byte type.
type Bytes = bencode.RawMessage

// AnnounceList is a list of the announces.
type AnnounceList [][]string

// Unique returns the list of the unique announces.
func (al AnnounceList) Unique() (announces []string) {
	announces = make([]string, 0, len(al))
	for _, tier := range al {
		for _, v := range tier {
			if v != "" && !helper.ContainsString(announces, v) {
				announces = append(announces, v)
			}
		}
	}
	return
}

// URLList represents a list of the url.
//
// BEP 19
type URLList []string

// FullURL returns the index-th full url.
//
// For the single-file case, name is the "name" of "info".
// For the multi-file case, name is the path "name/path/file"
// from "info" and "files".
//
// See http://bittorrent.org/beps/bep_0019.html
func (us URLList) FullURL(index int, name string) (url string) {
	if url = us[index]; strings.HasSuffix(url, "/") {
		url += name
	}
	return
}

// MarshalBencode implements the interface bencode.Marshaler.
func (us URLList) MarshalBencode() (b []byte, err error) {
	return bencode.EncodeBytes([]string(us))
}

// UnmarshalBencode implements the interface bencode.Unmarshaler.
func (us *URLList) UnmarshalBencode(b []byte) (err error) {
	var v interface{}
	if err = bencode.DecodeBytes(b, &v); err == nil {
		switch vs := v.(type) {
		case string:
			*us = URLList{vs}
		case []interface{}:
			urls := make(URLList, len(vs))
			for i, u := range vs {
				s, ok := u.(string)
				if !ok {
					return errors.New("the element of 'url-list' is not string")
				}
				urls[i] = s
			}
			*us = urls
		default:
			return errors.New("invalid 'url-lsit'")
		}
	}
	return
}

// MetaInfo represents the .torrent file.
type MetaInfo struct {
	InfoBytes    Bytes        `bencode:"info"`                    // BEP 3
	Announce     string       `bencode:"announce,omitempty"`      // BEP 3
	AnnounceList AnnounceList `bencode:"announce-list,omitempty"` // BEP 12
	Nodes        []HostAddr   `bencode:"nodes,omitempty"`         // BEP 5
	URLList      URLList      `bencode:"url-list,omitempty"`      // BEP 19

	// Where's this specified?
	// Mentioned at https://wiki.theory.org/index.php/BitTorrentSpecification.
	// All of them are optional.

	// CreationDate is the creation time of the torrent, in standard UNIX epoch
	// format (seconds since 1-Jan-1970 00:00:00 UTC).
	CreationDate int64 `bencode:"creation date,omitempty"`
	// Comment is the free-form textual comments of the author.
	Comment string `bencode:"comment,omitempty"`
	// CreatedBy is name and version of the program used to create the .torrent.
	CreatedBy string `bencode:"created by,omitempty"`
	// Encoding is the string encoding format used to generate the pieces part
	// of the info dictionary in the .torrent metafile.
	Encoding string `bencode:"encoding,omitempty"`
}

// Load loads a MetaInfo from an io.Reader.
func Load(r io.Reader) (mi MetaInfo, err error) {
	err = bencode.NewDecoder(r).Decode(&mi)
	return
}

// LoadFromFile loads a MetaInfo from a file.
func LoadFromFile(filename string) (mi MetaInfo, err error) {
	f, err := os.Open(filename)
	if err == nil {
		defer f.Close()
		mi, err = Load(f)
	}
	return
}

// Announces returns all the announces.
func (mi MetaInfo) Announces() AnnounceList {
	if len(mi.AnnounceList) > 0 {
		return mi.AnnounceList
	} else if mi.Announce != "" {
		return [][]string{{mi.Announce}}
	}
	return nil
}

// Magnet creates a Magnet from a MetaInfo.
//
// If displayName or infoHash is empty, it will be got from the info part.
func (mi MetaInfo) Magnet(displayName string, infoHash Hash) (m Magnet) {
	for _, t := range mi.Announces().Unique() {
		m.Trackers = append(m.Trackers, t)
	}

	if displayName == "" {
		info, _ := mi.Info()
		displayName = info.Name
	}

	if infoHash.IsZero() {
		infoHash = mi.InfoHash()
	}

	m.DisplayName = displayName
	m.InfoHash = infoHash
	return
}

// Write encodes the metainfo to w.
func (mi MetaInfo) Write(w io.Writer) error {
	return bencode.NewEncoder(w).Encode(mi)
}

// InfoHash returns the hash of the info.
func (mi MetaInfo) InfoHash() Hash {
	return NewHashFromBytes(mi.InfoBytes)
}

// Info parses the InfoBytes to the Info.
func (mi MetaInfo) Info() (info Info, err error) {
	err = bencode.DecodeBytes(mi.InfoBytes, &info)
	return
}
