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

import "fmt"

// ProtocolHeader is the BT protocal prefix.
//
// BEP 3
const ProtocolHeader = "\x13BitTorrent protocol"

// Predefine some message types.
const (
	// BEP 3
	MTypeChoke         MessageType = 0
	MTypeUnchoke       MessageType = 1
	MTypeInterested    MessageType = 2
	MTypeNotInterested MessageType = 3
	MTypeHave          MessageType = 4
	MTypeBitField      MessageType = 5
	MTypeRequest       MessageType = 6
	MTypePiece         MessageType = 7
	MTypeCancel        MessageType = 8

	// BEP 5
	MTypePort MessageType = 9

	// BEP 6 - Fast extension
	MTypeSuggest     MessageType = 0x0d // 13
	MTypeHaveAll     MessageType = 0x0e // 14
	MTypeHaveNone    MessageType = 0x0f // 15
	MTypeReject      MessageType = 0x10 // 16
	MTypeAllowedFast MessageType = 0x11 // 17

	// BEP 10
	MTypeExtended MessageType = 20
)

// MessageType is used to represent the message type.
type MessageType byte

func (mt MessageType) String() string {
	switch mt {
	case MTypeChoke:
		return "Choke"
	case MTypeUnchoke:
		return "Unchoke"
	case MTypeInterested:
		return "Interested"
	case MTypeNotInterested:
		return "NotInterested"
	case MTypeHave:
		return "Have"
	case MTypeBitField:
		return "Bitfield"
	case MTypeRequest:
		return "Request"
	case MTypePiece:
		return "Piece"
	case MTypeCancel:
		return "Cancel"
	case MTypePort:
		return "Port"
	case MTypeSuggest:
		return "Suggest"
	case MTypeHaveAll:
		return "HaveAll"
	case MTypeHaveNone:
		return "HaveNone"
	case MTypeReject:
		return "Reject"
	case MTypeAllowedFast:
		return "AllowedFast"
	case MTypeExtended:
		return "Extended"
	}
	return fmt.Sprintf("MessageType(%d)", mt)
}

// FastExtension reports whether the message type is fast extension.
func (mt MessageType) FastExtension() bool {
	return mt >= MTypeSuggest && mt <= MTypeAllowedFast
}
