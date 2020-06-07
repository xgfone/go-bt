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

package dht

import (
	"time"

	"github.com/xgfone/bt/krpc"
	"github.com/xgfone/bt/metainfo"
)

// RoutingTableNode represents the node with last changed time in the routing table.
type RoutingTableNode struct {
	Node        krpc.Node
	LastChanged time.Time
}

// RoutingTableStorage is used to store the nodes in the routing table.
type RoutingTableStorage interface {
	Load(ownid metainfo.Hash, ipv6 bool) (nodes []RoutingTableNode, err error)
	Dump(ownid metainfo.Hash, nodes []RoutingTableNode, ipv6 bool) (err error)
}

// NewNoopRoutingTableStorage returns a no-op RoutingTableStorage.
func NewNoopRoutingTableStorage() RoutingTableStorage { return noopStorage{} }

type noopStorage struct{}

func (s noopStorage) Load(metainfo.Hash, bool) (nodes []RoutingTableNode, err error) { return }
func (s noopStorage) Dump(metainfo.Hash, []RoutingTableNode, bool) (err error)       { return }
