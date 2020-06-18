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

package peerprotocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
)

var errMessageTooLong = fmt.Errorf("the peer message is too long")

// BitField represents the bit field of the pieces.
type BitField []uint8

// NewBitField returns a new BitField to hold the pieceNum pieces.
func NewBitField(pieceNum int) BitField {
	return make(BitField, (pieceNum+7)/8)
}

// NewBitFieldFromBools returns a new BitField from the bool list.
func NewBitFieldFromBools(bs []bool) BitField {
	bf := NewBitField(len(bs))
	for i, has := range bs {
		if has {
			bf.Set(uint32(i))
		}
	}
	return bf
}

// Bools converts the bit field to []bool.
func (bf BitField) Bools() []bool {
	bs := make([]bool, 0, len(bf)*8)
	for _, b := range bf {
		for i := 7; i >= 0; i-- {
			bs = append(bs, (b>>byte(i))&1 == 1)
		}
	}
	return bs
}

// Set sets the bit of the piece to 1 by its index.
func (bf BitField) Set(index uint32) {
	bf[index/8] |= (1 << byte(7-index%8))
}

// Unset sets the bit of the piece to 0 by its index.
func (bf BitField) Unset(index uint32) {
	bf[index/8] &^= (1 << byte(7-index%8))
}

// IsSet reports whether the bit of the piece is set to 1.
func (bf BitField) IsSet(index uint32) bool {
	return bf[index/8]&(1<<byte(7-index%8)) != 0
}

// Message is the message used by the peer protocol, which contains
// all the fields specified by the standard message types.
type Message struct {
	Keepalive bool
	Type      MessageType

	// Index is used by these message types:
	//
	//   BEP 3: Cancel, Request, Have, Piece
	//   BEP 6: Reject, Suggest, AllowedFast
	//
	Index uint32

	// Begin is used by these message types:
	//
	//   BEP 3: Request, Cancel, Piece
	//   BEP 6: Reject
	//
	Begin uint32

	// Length is used by these message types:
	//
	//   BEP 3: Request, Cancel
	//   BEP 6: Reject
	//
	Length uint32

	// Piece is used by these message types:
	//
	//   BEP 3: Piece
	Piece []byte

	// BitField is used by these message types:
	//
	//   BEP 3: Bitfield
	BitField BitField

	// ExtendedID and ExtendedPayload are used by these message types:
	//
	//   BEP 10: Extended
	//
	ExtendedID      uint8
	ExtendedPayload []byte

	// Port is used by these message types:
	//
	//   BEP 5: Port
	//
	Port uint16

	// UnknownTypePayload is the payload of the unknown message type.
	UnknownTypePayload []byte
}

// DecodeToMessage is equal to msg.Decode(r, maxLength).
func DecodeToMessage(r io.Reader, maxLength uint32) (msg Message, err error) {
	err = msg.Decode(r, maxLength)
	return
}

// UtMetadataExtendedMsg decodes the extended payload as UtMetadataExtendedMsg.
//
// Notice: the message type must be Extended.
func (m Message) UtMetadataExtendedMsg() (um UtMetadataExtendedMsg, err error) {
	if m.Type != Extended {
		panic("the message type is Extended")
	}
	err = um.DecodeFromPayload(m.ExtendedPayload)
	return
}

// UnmarshalBinary implements the interface encoding.BinaryUnmarshaler,
// which is equal to m.Decode(bytes.NewBuffer(data), 0).
func (m *Message) UnmarshalBinary(data []byte) (err error) {
	return m.Decode(bytes.NewBuffer(data), 0)
}

func readByte(r io.Reader) (b byte, err error) {
	var bs [1]byte
	if _, err = r.Read(bs[:]); err == nil {
		b = bs[0]
	}
	return
}

// Decode reads the data from r and decodes it to Message.
//
// if maxLength is equal to 0, it is unlimited. Or, it will read maxLength bytes
// at most.
func (m *Message) Decode(r io.Reader, maxLength uint32) (err error) {
	var length uint32
	if err = binary.Read(r, binary.BigEndian, &length); err != nil {
		if err != io.EOF {
			err = fmt.Errorf("error reading peer message message length: %s", err)
		}
		return
	}

	if length == 0 {
		m.Keepalive = true
		return
	} else if maxLength > 0 && length > maxLength {
		return errMessageTooLong
	}

	m.Keepalive = false
	lr := &io.LimitedReader{R: r, N: int64(length)}

	// Check that all of r was utilized.
	defer func() {
		if err == nil && lr.N != 0 {
			err = fmt.Errorf("%d bytes unused in message type %d", lr.N, m.Type)
		}
	}()

	_type, err := readByte(lr)
	if err != nil {
		return
	}

	switch m.Type = MessageType(_type); m.Type {
	case Choke, Unchoke, Interested, NotInterested, HaveAll, HaveNone:
	case Have, AllowedFast, Suggest:
		err = binary.Read(lr, binary.BigEndian, &m.Index)
	case Request, Cancel, Reject:
		if err = binary.Read(lr, binary.BigEndian, &m.Index); err != nil {
			return
		}
		if err = binary.Read(lr, binary.BigEndian, &m.Begin); err != nil {
			return
		}
		if err = binary.Read(lr, binary.BigEndian, &m.Length); err != nil {
			return
		}
	case Bitfield:
		_len := length - 1
		bs := make([]byte, _len)
		if _, err = io.ReadFull(lr, bs); err == nil {
			m.BitField = BitField(bs)
		}
	case Piece:
		if err = binary.Read(lr, binary.BigEndian, &m.Index); err != nil {
			return
		}
		if err = binary.Read(lr, binary.BigEndian, &m.Begin); err != nil {
			return
		}

		// TODO: Should we use a []byte pool?
		m.Piece = make([]byte, lr.N)
		if _, err = io.ReadFull(lr, m.Piece); err != nil {
			return fmt.Errorf("reading piece data error: %s", err)
		}
	case Extended:
		if m.ExtendedID, err = readByte(lr); err == nil {
			m.ExtendedPayload, err = ioutil.ReadAll(lr)
		}
	case Port:
		err = binary.Read(lr, binary.BigEndian, &m.Port)
	default:
		// err = fmt.Errorf("unknown message type %v", m.Type)
		m.UnknownTypePayload, err = ioutil.ReadAll(lr)
	}

	return
}

// MarshalBinary implements the interface encoding.BinaryMarshaler.
func (m Message) MarshalBinary() (data []byte, err error) {
	// TODO: Should we use a buffer pool?
	buf := bytes.NewBuffer(make([]byte, 0, 4))
	if err = m.Encode(buf); err == nil {
		data = buf.Bytes()
	}
	return
}

// Encode encodes the message to buf.
func (m Message) Encode(buf *bytes.Buffer) (err error) {
	// The 4-bytes is the placeholder of the length.
	buf.Reset()
	buf.Write([]byte{0, 0, 0, 0})

	// Write the non-keepalive message.
	if !m.Keepalive {
		if err = buf.WriteByte(byte(m.Type)); err != nil {
			return
		} else if err = m.marshalBinaryType(buf); err != nil {
			return
		}

		// Calculate and reset the length of the message body.
		data := buf.Bytes()
		if payloadLen := len(data) - 4; payloadLen > 0 {
			binary.BigEndian.PutUint32(data[:4], uint32(payloadLen))
		}
	}

	return
}

func (m Message) marshalBinaryType(buf *bytes.Buffer) (err error) {
	switch m.Type {
	case Choke, Unchoke, Interested, NotInterested, HaveAll, HaveNone:
	case Have:
		err = binary.Write(buf, binary.BigEndian, m.Index)
	case Request, Cancel, Reject:
		if err = binary.Write(buf, binary.BigEndian, m.Index); err != nil {
			return
		}
		if err = binary.Write(buf, binary.BigEndian, m.Begin); err != nil {
			return
		}
		if err = binary.Write(buf, binary.BigEndian, m.Length); err != nil {
			return
		}
	case Bitfield:
		buf.Write(m.BitField)
	case Piece:
		if err = binary.Write(buf, binary.BigEndian, m.Index); err != nil {
			return
		}
		if err = binary.Write(buf, binary.BigEndian, m.Begin); err != nil {
			return
		}
		_, err = buf.Write(m.Piece)
	case Extended:
		if err = buf.WriteByte(byte(m.ExtendedID)); err != nil {
			_, err = buf.Write(m.ExtendedPayload)
		}
	case Port:
		err = binary.Write(buf, binary.BigEndian, m.Port)
	default:
		// err = fmt.Errorf("unknown message type: %v", m.Type)
		_, err = buf.Write(m.UnknownTypePayload)
	}

	return
}
