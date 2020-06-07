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
	"fmt"
)

// ProtocolHeader is the BT protocal prefix.
//
// BEP 3
const ProtocolHeader = "\x13BitTorrent protocol"

// Predefine some message types.
const (
	// BEP 3
	Choke         MessageType = 0
	Unchoke       MessageType = 1
	Interested    MessageType = 2
	NotInterested MessageType = 3
	Have          MessageType = 4
	Bitfield      MessageType = 5
	Request       MessageType = 6
	Piece         MessageType = 7
	Cancel        MessageType = 8

	// BEP 5
	Port MessageType = 9

	// BEP 6 - Fast extension
	Suggest     MessageType = 0x0d // 13
	HaveAll     MessageType = 0x0e // 14
	HaveNone    MessageType = 0x0f // 15
	Reject      MessageType = 0x10 // 16
	AllowedFast MessageType = 0x11 // 17

	// BEP 10
	Extended MessageType = 20
)

// MessageType is used to represent the message type.
type MessageType byte

func (mt MessageType) String() string {
	switch mt {
	case Choke:
		return "Choke"
	case Unchoke:
		return "Unchoke"
	case Interested:
		return "Interested"
	case NotInterested:
		return "NotInterested"
	case Have:
		return "Have"
	case Bitfield:
		return "Bitfield"
	case Request:
		return "Request"
	case Piece:
		return "Piece"
	case Cancel:
		return "Cancel"
	case Port:
		return "Port"
	case Suggest:
		return "Suggest"
	case HaveAll:
		return "HaveAll"
	case HaveNone:
		return "HaveNone"
	case Reject:
		return "Reject"
	case AllowedFast:
		return "AllowedFast"
	case Extended:
		return "Extended"
	}
	return fmt.Sprintf("MessageType(%d)", mt)
}

// FastExtension reports whether the message type is fast extension.
func (mt MessageType) FastExtension() bool {
	return mt >= Suggest && mt <= AllowedFast
}
