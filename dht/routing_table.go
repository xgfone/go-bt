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

package dht

import (
	"sort"
	"sync"
	"time"

	"github.com/eyedeekay/go-i2p-bt/krpc"
	"github.com/eyedeekay/go-i2p-bt/metainfo"
)

const bktlen = 160

const (
	badTimeout     = time.Minute * 20
	dubiousTimeout = time.Minute * 15
)

type routingTable struct {
	k    int
	s    *Server
	ipv6 bool
	sync chan struct{}
	exit chan struct{}
	root metainfo.Hash
	lock sync.RWMutex
	bkts []*bucket
}

func newRoutingTable(s *Server, ipv6 bool) *routingTable {
	rt := &routingTable{
		k:    s.conf.K,
		s:    s,
		ipv6: ipv6,
		root: s.conf.ID,
		sync: make(chan struct{}),
		exit: make(chan struct{}),
		bkts: make([]*bucket, bktlen),
	}

	for i := range rt.bkts {
		rt.bkts[i] = &bucket{table: rt}
	}

	// Load all the nodes from the storage.
	nodes, err := s.conf.RoutingTableStorage.Load(s.conf.ID, ipv6)
	if err != nil {
		s.conf.ErrorLog("fail to load routing table(ipv6=%v): %s", ipv6, err)
	} else {
		now := time.Now()
		for _, node := range nodes {
			if now.Sub(node.LastChanged) < dubiousTimeout {
				rt.addNode(node.Node, node.LastChanged)
			}
		}
	}

	return rt
}

// Sync dumps the information of the routing table to the underlying storage.
func (rt *routingTable) Sync() {
	select {
	case rt.sync <- struct{}{}:
	case <-rt.exit:
	}
}

// Start starts the routing table.
func (rt *routingTable) Start(interval time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()

	for {
		select {
		case <-rt.exit:
			rt.dump()
			return
		case <-rt.sync:
			rt.dump()
		case now := <-tick.C:
			rt.lock.Lock()
			rt.checkAllBuckets(now)
			rt.lock.Unlock()
		}
	}
}

// Dump stores all the nodes into the underlying storage.
func (rt *routingTable) dump() {
	nodes := make([]RoutingTableNode, 0, 128)
	rt.lock.RLock()
	for _, bkt := range rt.bkts {
		for _, n := range bkt.Nodes {
			nodes = append(nodes, RoutingTableNode{
				Node:        krpc.Node{ID: n.ID, Addr: n.Addr},
				LastChanged: n.LastChanged,
			})
		}
	}
	rt.lock.RUnlock()

	if len(nodes) > 0 {
		err := rt.s.conf.RoutingTableStorage.Dump(rt.root, nodes, rt.ipv6)
		if err != nil {
			rt.s.conf.ErrorLog("fail to dump nodes in routing table(ipv6=%v): %s", rt.ipv6, err)
		}
	}
}

func (rt *routingTable) checkAllBuckets(now time.Time) {
	defer func() {
		if err := recover(); err != nil {
			rt.s.conf.ErrorLog("panic: %v", err)
		}
	}()

	for _, bkt := range rt.bkts {
		bkt.CheckAllNodes(now)
	}
}

func (rt *routingTable) Len() (n int) {
	rt.lock.RLock()
	for _, bkt := range rt.bkts {
		n += len(bkt.Nodes)
	}
	rt.lock.RUnlock()
	return
}

// Stop stops the routing table.
func (rt *routingTable) Stop() {
	select {
	case <-rt.exit:
	default:
		close(rt.exit)
	}
}

// AddNode adds the node into the routing table.
//
// The returned value:
//
//	NodeAdded:           The node is added successfully.
//	NodeNotAdded:        The node is not added and is discarded.
//	NodeExistAndUpdated: The node has existed, and its status has been updated.
//	NodeExistAndChanged: The node has existed, but the address is inconsistent.
//	                     The current node will be discarded.
func (rt *routingTable) AddNode(n krpc.Node) (r int) {
	if n.ID == rt.root { // Don't add itself.
		return NodeNotAdded
	}
	return rt.addNode(n, time.Now())
}

func (rt *routingTable) addNode(n krpc.Node, now time.Time) (r int) {
	bktid := bucketid(rt.root, n.ID)
	rt.lock.Lock()
	r = rt.bkts[bktid].AddNode(n, now)
	rt.lock.Unlock()
	return
}

// Closest returns the maxnum number of the nodes which are the closest to target.
func (rt *routingTable) Closest(target metainfo.Hash, maxnum int) (nodes []krpc.Node) {
	close := nodesByDistance{target: target, maxnum: maxnum}
	rt.lock.RLock()
	for _, b := range rt.bkts {
		for _, n := range b.Nodes {
			close.push(n.Node)
		}
	}
	rt.lock.RUnlock()
	return close.nodes
}

type nodesByDistance struct {
	maxnum int
	target metainfo.Hash
	nodes  []krpc.Node
}

func (ns *nodesByDistance) push(node krpc.Node) {
	ix := sort.Search(len(ns.nodes), func(i int) bool {
		return distcmp(ns.target, ns.nodes[i].ID, node.ID) > 0
	})

	if len(ns.nodes) < ns.maxnum {
		ns.nodes = append(ns.nodes, node)
	}

	// slide existing nodes down to make room.
	// this will overwrite the entry we just appended.
	if ix != len(ns.nodes) {
		copy(ns.nodes[ix+1:], ns.nodes[ix:])
		ns.nodes[ix] = node
	}
}

// distcmp compares the distances a->target and b->target.
// Returns -1 if a is closer to target, 1 if b is closer to target
// and 0 if they are equal.
func distcmp(target, a, b metainfo.Hash) int {
	for i := range target {
		da := a[i] ^ target[i]
		db := b[i] ^ target[i]
		if da > db {
			return 1
		} else if da < db {
			return -1
		}
	}
	return 0
}

/// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
/// Bucket

// Predefine some values returned by the method AddNode of routing table.
const (
	NodeAdded = iota
	NodeNotAdded
	NodeExistAndUpdated
	NodeExistAndChanged
)

type bucket struct {
	table       *routingTable
	Nodes       []*wrappedNode
	LastChanged time.Time
}

func (b *bucket) emit(f func(metainfo.Hash, RoutingTableNode) error, n *wrappedNode) {
	node := RoutingTableNode{Node: n.Node, LastChanged: n.LastChanged}
	f(b.table.root, node)
}

func (b *bucket) UpdateLastChangedTime(now time.Time) {
	b.LastChanged = now
}

func (b *bucket) AddNode(n krpc.Node, now time.Time) (status int) {
	// Update the old one.
	for _, orig := range b.Nodes {
		if orig.ID == n.ID {
			if orig.Addr.Equal(n.Addr) {
				orig.UpdateLastChangedTime(now)
				status = NodeExistAndUpdated
				return
			}

			// // TODO: Should we replace the old one??
			// b.UpdateLastChangedTime(now)
			// copy(b.Nodes[i:], b.Nodes[i+1:])
			// b.Nodes[len(b.Nodes)-1] = newWrappedNode(b, n, now)
			// status = nodeExistAndChanged
			return NodeExistAndChanged
		}
	}

	// The bucket is not full and append it.
	if _len := len(b.Nodes); _len < b.table.k {
		b.UpdateLastChangedTime(now)
		b.Nodes = append(b.Nodes, newWrappedNode(b, n, now))
		status = NodeAdded
		return
	}

	// The bucket is full and remove the bad node, or discard the current node.
	for i, node := range b.Nodes {
		if node.IsBad(now) {
			b.UpdateLastChangedTime(now)
			copy(b.Nodes[i:], b.Nodes[i+1:])
			b.Nodes = append(b.Nodes, newWrappedNode(b, n, now))
			status = NodeAdded
			return
		}
	}

	// It will discard the current node and return false.
	status = NodeNotAdded
	return
}

func (b *bucket) delNodeByIndex(index int) {
	b.Nodes = append(b.Nodes[:index], b.Nodes[index+1:]...)
}

func (b *bucket) CheckAllNodes(now time.Time) {
	_len := len(b.Nodes)
	if _len == 0 || now.Sub(b.LastChanged) < dubiousTimeout {
		return
	}

	// Check all the dubious nodes.
	indexes := make([]int, 0, len(b.Nodes))
	for i, node := range b.Nodes {
		switch status := node.Status(now); status {
		case nodeStatusGood:
		case nodeStatusDubious:
			// Try to send the PING query to the dubious node to check whether it is alive.
			if err := b.table.s.Ping(node.Node.Addr.Addr()); err != nil {
				b.table.s.conf.ErrorLog("fail to ping '%s': %s", node.Node.String(), err)
			}
		case nodeStatusBad:
			// Remove the bad node
			indexes = append(indexes, i)
		default:
			panic(status)
		}
	}

	// Remove all the bad nodes.
	for _, index := range indexes {
		b.delNodeByIndex(index)
	}
}

func bucketid(ownerid, nid metainfo.Hash) int {
	var i int
	var bite byte
	var bitDiff int
	var v byte
	for i, bite = range ownerid {
		v = bite ^ nid[i]
		switch {
		case v&0x80 > 0:
			bitDiff = 7
			goto calc
		case v&0x40 > 0:
			bitDiff = 6
			goto calc
		case v&0x20 > 0:
			bitDiff = 5
			goto calc
		case v&0x10 > 0:
			bitDiff = 4
			goto calc
		case v&0x08 > 0:
			bitDiff = 3
			goto calc
		case v&0x04 > 0:
			bitDiff = 2
			goto calc
		case v&0x02 > 0:
			bitDiff = 1
			goto calc
		case v&0x01 > 0:
			bitDiff = 0
			goto calc
		}
	}

calc:
	return i*8 + (7 - bitDiff)
}

/// >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
/// Node

const (
	nodeStatusGood = iota
	nodeStatusDubious
	nodeStatusBad
)

type wrappedNode struct {
	krpc.Node
	LastChanged time.Time
	bkt         *bucket
}

func newWrappedNode(bkt *bucket, n krpc.Node, now time.Time) *wrappedNode {
	return &wrappedNode{bkt: bkt, Node: n, LastChanged: now}
}

func (n *wrappedNode) UpdateLastChangedTime(now ...time.Time) {
	var _now time.Time
	if len(now) > 0 {
		_now = now[0]
	} else {
		_now = time.Now()
	}

	n.LastChanged = _now
	n.bkt.UpdateLastChangedTime(_now)
	return
}

func (n *wrappedNode) IsBad(now ...time.Time) bool {
	var _now time.Time
	if len(now) > 0 {
		_now = now[0]
	} else {
		_now = time.Now()
	}
	return _now.Sub(n.LastChanged) > badTimeout
}

func (n *wrappedNode) IsDubious(now ...time.Time) bool {
	var _now time.Time
	if len(now) > 0 {
		_now = now[0]
	} else {
		_now = time.Now()
	}
	return _now.Sub(n.LastChanged) > dubiousTimeout
}

func (n *wrappedNode) Status(now time.Time) int {
	duration := now.Sub(n.LastChanged)
	switch {
	case duration > badTimeout:
		return nodeStatusBad
	case duration > dubiousTimeout:
		return nodeStatusDubious
	default:
		return nodeStatusGood
	}
}
