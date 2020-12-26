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
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/xgfone/bt/bencode"
)

var zeroHash Hash

// HashSize is the size of the InfoHash.
const HashSize = 20

// Hash is the 20-byte SHA1 hash used for info and pieces.
type Hash [HashSize]byte

// NewRandomHash returns a random hash.
func NewRandomHash() (h Hash) {
	rand.Read(h[:])
	return
}

// NewHash converts the 20-bytes to Hash.
func NewHash(b []byte) (h Hash) {
	copy(h[:], b[:HashSize])
	return
}

// NewHashFromString returns a new Hash from a string.
func NewHashFromString(s string) (h Hash) {
	err := h.FromString(s)
	if err != nil {
		panic(err)
	}
	return
}

// NewHashFromHexString returns a new Hash from a hex string.
func NewHashFromHexString(s string) (h Hash) {
	err := h.FromHexString(s)
	if err != nil {
		panic(err)
	}
	return
}

// NewHashFromBytes returns a new Hash from a byte slice.
func NewHashFromBytes(b []byte) (ret Hash) {
	hasher := sha1.New()
	hasher.Write(b)
	copy(ret[:], hasher.Sum(nil))
	return
}

// Bytes returns the byte slice type.
func (h Hash) Bytes() []byte {
	return h[:]
}

// String is equal to HexString.
func (h Hash) String() string {
	return h.HexString()
}

// BytesString returns the bytes string, that's, string(h[:]).
func (h Hash) BytesString() string {
	return string(h[:])
}

// HexString returns the hex string format.
func (h Hash) HexString() string {
	return hex.EncodeToString(h[:])
}

// IsZero reports whether the whole hash is zero.
func (h Hash) IsZero() bool {
	return h == zeroHash
}

// WriteBinary is the same as MarshalBinary, but writes the result into w
// instead of returning.
func (h Hash) WriteBinary(w io.Writer) (m int, err error) {
	return w.Write(h[:])
}

// UnmarshalBinary implements the interface binary.BinaryUnmarshaler.
func (h *Hash) UnmarshalBinary(b []byte) (err error) {
	if len(b) < HashSize {
		return errors.New("Hash.UnmarshalBinary: too few bytes")
	}
	copy((*h)[:], b[:HashSize])
	return
}

// MarshalBinary implements the interface binary.BinaryMarshaler.
func (h Hash) MarshalBinary() (data []byte, err error) {
	return h[:], nil
}

// MarshalBencode implements the interface bencode.Marshaler.
func (h Hash) MarshalBencode() (b []byte, err error) {
	return bencode.EncodeBytes(h[:])
}

// UnmarshalBencode implements the interface bencode.Unmarshaler.
func (h *Hash) UnmarshalBencode(b []byte) (err error) {
	var s string
	if err = bencode.NewDecoder(bytes.NewBuffer(b)).Decode(&s); err == nil {
		err = h.FromString(s)
	}
	return
}

// FromString resets the info hash from the string.
func (h *Hash) FromString(s string) (err error) {
	switch len(s) {
	case HashSize:
		copy(h[:], s)
	case 2 * HashSize:
		err = h.FromHexString(s)
	case 32:
		var bs []byte
		if bs, err = base32.StdEncoding.DecodeString(s); err == nil {
			copy(h[:], bs)
		}
	default:
		err = fmt.Errorf("hash string has bad length: %d", len(s))
	}

	return nil
}

// FromHexString resets the info hash from the hex string.
func (h *Hash) FromHexString(s string) (err error) {
	if len(s) != 2*HashSize {
		err = fmt.Errorf("hash hex string has bad length: %d", len(s))
		return
	}

	n, err := hex.Decode(h[:], []byte(s))
	if err != nil {
		return
	}

	if n != HashSize {
		panic(n)
	}
	return
}

// Xor returns the hash of h XOR o.
func (h Hash) Xor(o Hash) (ret Hash) {
	for i := range o {
		ret[i] = h[i] ^ o[i]
	}
	return
}

// Compare returns 0 if h == o, -1 if h < o, or +1 if h > o.
func (h Hash) Compare(o Hash) int { return bytes.Compare(h[:], o[:]) }

/// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

// Hashes is a set of Hashes.
type Hashes []Hash

// Contains reports whether hs contains h.
func (hs Hashes) Contains(h Hash) bool {
	for _, _h := range hs {
		if h == _h {
			return true
		}
	}
	return false
}

// MarshalBencode implements the interface bencode.Marshaler.
func (hs Hashes) MarshalBencode() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(HashSize * len(hs))
	for _, h := range hs {
		buf.Write(h[:])
	}
	return bencode.EncodeBytes(buf.Bytes())
}

// UnmarshalBencode implements the interface bencode.Unmarshaler.
func (hs *Hashes) UnmarshalBencode(b []byte) (err error) {
	var bs []byte
	if err = bencode.DecodeBytes(b, &bs); err != nil {
		return
	}

	_len := len(bs)
	if _len%HashSize != 0 {
		return fmt.Errorf("Hashes: invalid bytes length '%d'", _len)
	}

	hashes := make(Hashes, 0, _len/HashSize)
	for i := 0; i < _len; i += HashSize {
		var h Hash
		copy(h[:], bs[i:i+HashSize])
		hashes = append(hashes, h)
	}

	*hs = hashes
	return
}
