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

// Package dht implements the DHT Protocol. And you can use it to build or join
// the DHT swarm network.
package dht

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/xgfone/bt/bencode"
	"github.com/xgfone/bt/krpc"
	"github.com/xgfone/bt/metainfo"
	"github.com/xgfone/bt/utils"
)

const (
	queryMethodPing         = "ping"
	queryMethodFindNode     = "find_node"
	queryMethodGetPeers     = "get_peers"
	queryMethodAnnouncePeer = "announce_peer"
)

var errUnsupportedIPProtocol = fmt.Errorf("unsupported ip protocol")

// Predefine some ip protocol stacks.
const (
	IPv4Protocol IPProtocolStack = 4
	IPv6Protocol IPProtocolStack = 6
)

// IPProtocolStack represents the ip protocol stack, such as IPv4 or IPv6
type IPProtocolStack uint8

// Result is used to pass the response result to the callback function.
type Result struct {
	// Addr is the address of the peer where the request is sent to.
	//
	// Notice: it may be nil for "get_peers" request.
	Addr net.Addr

	// For Error
	Code    int    // 0 represents the success.
	Reason  string // Reason indicates the reason why the request failed when Code > 0.
	Timeout bool   // Timeout indicates whether the response is timeout.

	// The list of the address of the peers returned by GetPeers.
	Peers []metainfo.Address
}

// Config is used to configure the DHT server.
type Config struct {
	// K is the size of the bucket of the routing table.
	//
	// The default is 8.
	K int

	// ID is the id of the current DHT server node.
	//
	// The default is generated randomly
	ID metainfo.Hash

	// IPProtocols is used to specify the supported IP Protocol Stack.
	//
	// If not given, it will detect the address family of the server listens
	// automatically and set the supported ip protocol stack by the address
	// family. If fails, the default is []IPProtocolStack{IPv4Protocol}.
	// For the empty address, it supports the ipv4/ipv6 protocols by default.
	// So, if the server is listening on the empty address, such as ":6881",
	// and only supports IPv6, you should specify it to "IPv6Protocol".
	IPProtocols []IPProtocolStack

	// ReadOnly indicates whether the current node is read-only.
	//
	// If true, the DHT server will enter in the read-only mode.
	//
	// The default is false.
	//
	// BEP 43
	ReadOnly bool

	// MsgSize is the maximum size of the DHT message.
	//
	// The default is 4096.
	MsgSize int

	// SearchDepth is used to control the depth to send the "get_peers" or
	// "find_node" query recursively to get the peers storing the torrent
	// infohash.
	//
	// The default depth is 8.
	SearchDepth int

	// RespTimeout is the response timeout, that's, the response is valid
	// only before the timeout reaches.
	//
	// The default is "10s".
	RespTimeout time.Duration

	// RoutingTableStorage is used to store the nodes in the routing table.
	//
	// The default is nil.
	RoutingTableStorage RoutingTableStorage

	// PeerManager is used to manage the peers on torrent infohash,
	// which is called when receiving the "get_peers" query.
	//
	// The default uses the inner token-peer manager.
	PeerManager PeerManager

	// Blacklist is used to manage the ip blacklist.
	//
	// The default is NewMemoryBlacklist(1024, time.Hour*24*7).
	Blacklist Blacklist

	// ErrorLog is used to log the error.
	//
	// The default is log.Printf.
	ErrorLog func(format string, args ...interface{})

	// OnSearch is called when someone searches the torrent infohash,
	// that's, the "get_peers" query.
	//
	// The default callback does noting.
	OnSearch func(infohash string, ip net.Addr)

	// OnTorrent is called when someone has the torrent infohash
	// or someone has just downloaded the torrent infohash,
	// that's, the "get_peers" response or "announce_peer" query.
	//
	// The default callback does noting.
	OnTorrent func(infohash string, ip net.Addr)

	// HandleInMessage is used to intercept the incoming DHT message.
	// For example, you can debug the message as the log.
	//
	// Return true if going on handling by the default. Or return false.
	//
	// The default is nil.
	HandleInMessage func(net.Addr, *krpc.Message) bool

	// HandleOutMessage is used to intercept the outgoing DHT message.
	// For example, you can debug the message as the log.
	//
	// Return (false, nil) if going on handling by the default.
	//
	// The default is nil.
	HandleOutMessage func(net.Addr, *krpc.Message) (wrote bool, err error)
}

func (c Config) in(net.Addr, *krpc.Message) bool           { return true }
func (c Config) out(net.Addr, *krpc.Message) (bool, error) { return false, nil }

func (c *Config) set(conf ...Config) {
	if len(conf) > 0 {
		*c = conf[0]
	}

	if c.K <= 0 {
		c.K = 8
	}
	if c.ID.IsZero() {
		c.ID = metainfo.NewRandomHash()
	}
	if c.MsgSize <= 0 {
		c.MsgSize = 4096
	}
	if c.ErrorLog == nil {
		c.ErrorLog = log.Printf
	}
	if c.SearchDepth < 1 {
		c.SearchDepth = 8
	}
	if c.RoutingTableStorage == nil {
		c.RoutingTableStorage = noopStorage{}
	}
	if c.Blacklist == nil {
		c.Blacklist = NewMemoryBlacklist(1024, time.Hour*24*7)
	}
	if c.RespTimeout == 0 {
		c.RespTimeout = time.Second * 10
	}
	if c.OnSearch == nil {
		c.OnSearch = func(string, net.Addr) {}
	}
	if c.OnTorrent == nil {
		c.OnTorrent = func(string, net.Addr) {}
	}
	if c.HandleInMessage == nil {
		c.HandleInMessage = c.in
	}
	if c.HandleOutMessage == nil {
		c.HandleOutMessage = c.out
	}
}

// Server is a DHT server.
type Server struct {
	conf Config
	exit chan struct{}
	conn net.PacketConn
	once sync.Once

	ipv4 bool
	ipv6 bool
	want []krpc.Want

	peerManager        PeerManager
	routingTable4      *routingTable
	routingTable6      *routingTable
	tokenManager       *tokenManager
	tokenPeerManager   *tokenPeerManager
	transactionManager *transactionManager
}

// NewServer returns a new DHT server.
func NewServer(conn net.PacketConn, config ...Config) *Server {
	var conf Config
	conf.set(config...)

	if len(conf.IPProtocols) == 0 {
		host, _, err := net.SplitHostPort(conn.LocalAddr().String())
		if err != nil {
			panic(err)
		} else if ip := net.ParseIP(host); utils.IpIsZero(ip) {
			conf.IPProtocols = []IPProtocolStack{IPv4Protocol, IPv6Protocol}
		} else if ip.To4() != nil {
			conf.IPProtocols = []IPProtocolStack{IPv4Protocol}
		} else {
			conf.IPProtocols = []IPProtocolStack{IPv6Protocol}
		}
	}

	var ipv4, ipv6 bool
	var want []krpc.Want
	for _, ip := range conf.IPProtocols {
		switch ip {
		case IPv4Protocol:
			ipv4 = true
			want = append(want, krpc.WantNodes)
		case IPv6Protocol:
			ipv6 = true
			want = append(want, krpc.WantNodes6)
		}
	}

	s := &Server{
		ipv4:               ipv4,
		ipv6:               ipv6,
		want:               want,
		conn:               conn,
		conf:               conf,
		exit:               make(chan struct{}),
		peerManager:        conf.PeerManager,
		tokenManager:       newTokenManager(),
		tokenPeerManager:   newTokenPeerManager(),
		transactionManager: newTransactionManager(),
	}

	s.routingTable4 = newRoutingTable(s, false)
	s.routingTable6 = newRoutingTable(s, true)
	if s.peerManager == nil {
		s.peerManager = s.tokenPeerManager
	}

	return s
}

// ID returns the ID of the DHT server node.
func (s *Server) ID() metainfo.Hash { return s.conf.ID }

// Bootstrap initializes the routing table at first.
//
// Notice: If the routing table has had some nodes, it does noting.
func (s *Server) Bootstrap(addrs []string) {
	if (s.ipv4 && s.routingTable4.Len() == 0) ||
		(s.ipv6 && s.routingTable6.Len() == 0) {
		for _, addr := range addrs {
			as, err := metainfo.NewAddressesFromString(addr)
			if err != nil {
				s.conf.ErrorLog(err.Error())
				continue
			}

			for _, a := range as {
				if err = s.FindNode(a.Addr(), s.conf.ID); err != nil {
					s.conf.ErrorLog(`fail to bootstrap '%s': %s`, a.String(), err)
				}
			}
		}
	}
}

// Node4Num returns the number of the ipv4 nodes in the routing table.
func (s *Server) Node4Num() int { return s.routingTable4.Len() }

// Node6Num returns the number of the ipv6 nodes in the routing table.
func (s *Server) Node6Num() int { return s.routingTable6.Len() }

// AddNode adds the node into the routing table.
//
// The returned value:
//   NodeAdded:           The node is added successfully.
//   NodeNotAdded:        The node is not added and is discarded.
//   NodeExistAndUpdated: The node has existed, and its status has been updated.
//   NodeExistAndChanged: The node has existed, but the address is inconsistent.
//                        The current node will be discarded.
//
func (s *Server) AddNode(node krpc.Node) int {
	// For IPv6
	if node.Addr.IsIPv6() {
		if s.ipv6 {
			return s.routingTable6.AddNode(node)
		}
		return NodeNotAdded
	}

	// For IPv4
	if s.ipv4 {
		return s.routingTable4.AddNode(node)
	}

	return NodeNotAdded
}

func (s *Server) addNode(a net.Addr, id metainfo.Hash, ro bool) (r int) {
	if ro { // BEP 43
		return NodeNotAdded
	}

	if r = s.AddNode(krpc.NewNodeByUDPAddr(id, a)); r == NodeExistAndChanged {
		s.conf.Blacklist.Add(utils.IPAddr(a), utils.Port(a))
	}

	return
}

func (s *Server) addNode2(node krpc.Node, ro bool) int {
	if ro { // BEP 43
		return NodeNotAdded
	}
	return s.AddNode(node)
}

func (s *Server) stop() {
	close(s.exit)
	s.routingTable4.Stop()
	s.routingTable6.Stop()
	s.tokenManager.Stop()
	s.tokenPeerManager.Stop()
	s.transactionManager.Stop()
	s.conf.Blacklist.Close()
}

// Close stops the DHT server.
func (s *Server) Close() { s.once.Do(s.stop) }

// Sync is used to synchronize the routing table to the underlying storage.
func (s *Server) Sync() {
	s.routingTable4.Sync()
	s.routingTable6.Sync()
}

// Run starts the DHT server.
func (s *Server) Run() {
	go s.routingTable4.Start(time.Minute * 5)
	go s.routingTable6.Start(time.Minute * 5)
	go s.tokenManager.Start(time.Minute * 10)
	go s.tokenPeerManager.Start(time.Hour * 24)
	go s.transactionManager.Start(s, s.conf.RespTimeout)

	buf := make([]byte, s.conf.MsgSize)
	for {
		n, raddr, err := s.conn.ReadFrom(buf)
		if err != nil {
			s.conf.ErrorLog("fail to read the dht message: %s", err)
			return
		}

		s.handlePacket(raddr.(net.Addr), buf[:n])
	}
}

func (s *Server) isDisabled(raddr net.Addr) bool {
	if utils.IsIPv6Addr(raddr) {
		if !s.ipv6 {
			return true
		}
	} else if !s.ipv4 {
		return true
	}
	return false
}

// HandlePacket handles the incoming DHT message.
func (s *Server) handlePacket(raddr net.Addr, data []byte) {
	if s.isDisabled(raddr) {
		return
	}

	// Check whether the raddr is in the ip blacklist. If yes, discard it.
	if s.conf.Blacklist.In(utils.IPAddr(raddr), utils.Port(raddr)) {
		return
	}

	var msg krpc.Message
	if err := bencode.DecodeBytes(data, &msg); err != nil {
		s.conf.ErrorLog("decode krpc message error: %s", err)
		return
	} else if msg.T == "" {
		s.conf.ErrorLog("no transaction id from '%s'", raddr)
		return
	}

	// TODO: Should we use a task pool??
	go s.handleMessage(raddr, msg)
}

func (s *Server) handleMessage(raddr net.Addr, m krpc.Message) {
	if !s.conf.HandleInMessage(raddr, &m) {
		return
	}

	switch m.Y {
	case "q":
		if !m.A.ID.IsZero() {
			r := s.addNode(raddr, m.A.ID, m.RO)
			if r != NodeExistAndChanged && !s.conf.ReadOnly { // BEP 43
				s.handleQuery(raddr, m)
			}
		}
	case "r":
		if !m.R.ID.IsZero() {
			if s.addNode(raddr, m.R.ID, m.RO) == NodeExistAndChanged {
				return
			}

			if t := s.transactionManager.PopTransaction(m.T, raddr); t != nil {
				t.OnResponse(t, raddr, m)
			}
		}
	case "e":
		if t := s.transactionManager.PopTransaction(m.T, raddr); t != nil {
			t.OnError(t, m.E.Code, m.E.Reason)
		}
	default:
		s.conf.ErrorLog("unknown dht message type '%s'", m.Y)
	}
}

func (s *Server) handleQuery(raddr net.Addr, m krpc.Message) {
	switch m.Q {
	case queryMethodPing:
		s.reply(raddr, m.T, krpc.ResponseResult{})
	case queryMethodFindNode: // See BEP 32
		var r krpc.ResponseResult
		n4 := m.A.ContainsWant(krpc.WantNodes)
		n6 := m.A.ContainsWant(krpc.WantNodes6)
		if !n4 && !n6 {
			if utils.IsIPv6Addr(raddr) {
				r.Nodes6 = s.routingTable6.Closest(m.A.InfoHash, s.conf.K)
			} else {
				r.Nodes = s.routingTable4.Closest(m.A.InfoHash, s.conf.K)
			}
		} else {
			if n4 {
				r.Nodes = s.routingTable4.Closest(m.A.InfoHash, s.conf.K)
			}
			if n6 {
				r.Nodes6 = s.routingTable6.Closest(m.A.InfoHash, s.conf.K)
			}
		}
		s.reply(raddr, m.T, r)
	case queryMethodGetPeers: // See BEP 32
		n4 := m.A.ContainsWant(krpc.WantNodes)
		n6 := m.A.ContainsWant(krpc.WantNodes6)

		// Get the ipv4/ipv6 peers storing the torrent infohash.
		var r krpc.ResponseResult
		if !n4 && !n6 {
			r.Values = s.peerManager.GetPeers(m.A.InfoHash, s.conf.K, utils.IsIPv6Addr(raddr))
		} else {
			if n4 {
				r.Values = s.peerManager.GetPeers(m.A.InfoHash, s.conf.K, false)
			}

			if n6 {
				values := s.peerManager.GetPeers(m.A.InfoHash, s.conf.K, true)
				if len(r.Values) == 0 {
					r.Values = values
				} else {
					r.Values = append(r.Values, values...)
				}
			}
		}

		// No Peers, and return the closest other nodes.
		if len(r.Values) == 0 {
			if !n4 && !n6 {
				if utils.IsIPv6Addr(raddr) {
					r.Nodes6 = s.routingTable6.Closest(m.A.InfoHash, s.conf.K)
				} else {
					r.Nodes = s.routingTable4.Closest(m.A.InfoHash, s.conf.K)
				}
			} else {
				if n4 {
					r.Nodes = s.routingTable4.Closest(m.A.InfoHash, s.conf.K)
				}
				if n6 {
					r.Nodes6 = s.routingTable6.Closest(m.A.InfoHash, s.conf.K)
				}
			}
		}

		r.Token = s.tokenManager.Token(raddr)
		s.reply(raddr, m.T, r)
		s.conf.OnSearch(m.A.InfoHash.HexString(), raddr)
	case queryMethodAnnouncePeer:
		if s.tokenManager.Check(raddr, m.A.Token) {
			return
		}
		s.reply(raddr, m.T, krpc.ResponseResult{})
		s.conf.OnTorrent(m.A.InfoHash.HexString(), raddr)
	default:
		s.sendError(raddr, m.T, "unknown query method", krpc.ErrorCodeMethodUnknown)
	}
}

func (s *Server) send(raddr net.Addr, m krpc.Message) (wrote bool, err error) {
	// // TODO: Should we check the ip blacklist??
	// if s.conf.Blacklist.In(utils.IPaddr(raddr), utils.Port(raddr)) {
	//     return
	// }

	m.RO = s.conf.ReadOnly // BEP 43
	if wrote, err = s.conf.HandleOutMessage(raddr, &m); !wrote && err == nil {
		wrote, err = s._send(raddr, m)
	}

	return
}

func (s *Server) _send(raddr net.Addr, m krpc.Message) (wrote bool, err error) {
	if m.T == "" || m.Y == "" {
		panic(`DHT message "t" or "y" must not be empty`)
	}

	buf := bytes.NewBuffer(nil)
	buf.Grow(128)
	if err = bencode.NewEncoder(buf).Encode(m); err != nil {
		panic(err)
	}

	n, err := s.conn.WriteTo(buf.Bytes(), raddr)
	if err != nil {
		err = fmt.Errorf("error writing %d bytes to %s: %s", buf.Len(), raddr, err)
		s.conf.Blacklist.Add(utils.IPAddr(raddr), 0)
		return
	}

	wrote = true
	if n != buf.Len() {
		err = io.ErrShortWrite
	}

	return
}

func (s *Server) sendError(raddr net.Addr, tid, reason string, code int) {
	if _, err := s.send(raddr, krpc.NewErrorMsg(tid, code, reason)); err != nil {
		s.conf.ErrorLog("error replying to %s: %s", raddr.String(), err.Error())
	}
}

func (s *Server) reply(raddr net.Addr, tid string, r krpc.ResponseResult) {
	r.ID = s.conf.ID
	if _, err := s.send(raddr, krpc.NewResponseMsg(tid, r)); err != nil {
		s.conf.ErrorLog("error replying to %s: %s", raddr.String(), err.Error())
	}
}

func (s *Server) request(t *transaction) (err error) {
	if s.isDisabled(t.Addr) {
		return errUnsupportedIPProtocol
	}

	t.Arg.ID = s.conf.ID
	if t.ID == "" {
		t.ID = s.transactionManager.GetTransactionID()
	}
	if _, err = s.send(t.Addr, krpc.NewQueryMsg(t.ID, t.Query, t.Arg)); err != nil {
		s.conf.ErrorLog("error replying to %s: %s", t.Addr.String(), err.Error())
	} else {
		s.transactionManager.AddTransaction(t)
	}
	return
}

func (s *Server) onError(t *transaction, code int, reason string) {
	s.conf.ErrorLog("got an error to ping '%s': code=%d, reason=%s",
		t.Addr.String(), code, reason)
	t.Done(Result{Code: code, Reason: reason})
}

func (s *Server) onTimeout(t *transaction) {
	// TODO: Should we use a task pool??
	t.Done(Result{Timeout: true})
	s.conf.ErrorLog("transaction '%s' timeout: query=%s, raddr=%s",
		t.ID, t.Query, t.Addr.String())
}

func (s *Server) onPingResp(t *transaction, a net.Addr, m krpc.Message) {
	t.Done(Result{})
}

func (s *Server) onGetPeersResp(t *transaction, a net.Addr, m krpc.Message) {
	// Store the response node with the token.
	if m.R.Token != "" {
		s.tokenPeerManager.Set(m.R.ID, a, m.R.Token)
	}

	// Get the peers.
	if len(m.R.Values) > 0 {
		t.Done(Result{Peers: m.R.Values})
		for _, addr := range m.R.Values {
			s.conf.OnTorrent(t.Arg.InfoHash.HexString(), addr.IP)
		}
		return
	}

	// Search the torrent infohash recursively.
	t.Depth--
	if t.Depth < 1 {
		t.Done(Result{})
		return
	}

	var found bool
	ids := t.Visited
	nodes := make([]krpc.Node, 0, len(m.R.Nodes)+len(m.R.Nodes6))
	for _, node := range m.R.Nodes {
		if node.ID == t.Arg.InfoHash {
			found = true
		}
		if ids.Contains(node.ID) {
			continue
		}
		if s.addNode2(node, m.RO) == NodeAdded {
			nodes = append(nodes, node)
			ids = append(ids, node.ID)
		}
	}
	for _, node := range m.R.Nodes6 {
		if node.ID == t.Arg.InfoHash {
			found = true
		}
		if ids.Contains(node.ID) {
			continue
		}
		if s.addNode2(node, m.RO) == NodeAdded {
			nodes = append(nodes, node)
			ids = append(ids, node.ID)
		}
	}

	if found || len(nodes) == 0 {
		t.Done(Result{})
		return
	}

	for _, node := range nodes {
		s.getPeers(t.Arg.InfoHash, node.Addr, t.Depth, ids, t.Callback)
	}
}

func (s *Server) getPeers(info metainfo.Hash, addr metainfo.Address, depth int,
	ids metainfo.Hashes, cb ...func(Result)) {
	arg := krpc.QueryArg{InfoHash: info, Wants: s.want}
	t := newTransaction(s, addr.Addr(), queryMethodGetPeers, arg, cb...)
	t.OnResponse = s.onGetPeersResp
	t.Depth = depth
	t.Visited = ids
	if err := s.request(t); err != nil {
		s.conf.ErrorLog("fail to send query message to '%s': %s", addr.String(), err)
	}
}

// Ping sends a PING query to addr, and the callback function cb will be called
// when the response or error is returned, or it's timeout.
func (s *Server) Ping(addr net.Addr, cb ...func(Result)) (err error) {
	t := newTransaction(s, addr, queryMethodPing, krpc.QueryArg{}, cb...)
	t.OnResponse = s.onPingResp
	return s.request(t)
}

// GetPeers searches the peer storing the torrent by the infohash of the torrent,
// which will search it recursively until some peers are returned or it reaches
// the maximun depth, that's, ServerConfig.SearchDepth.
//
// If cb is given, it will be called when some peers are returned.
// Notice: it may be called for many times.
func (s *Server) GetPeers(infohash metainfo.Hash, cb ...func(Result)) {
	if infohash.IsZero() {
		panic("the infohash of the torrent is ZERO")
	}

	var nodes []krpc.Node
	if s.ipv4 {
		nodes = s.routingTable4.Closest(infohash, s.conf.K)
	}
	if s.ipv6 {
		nodes = append(nodes, s.routingTable6.Closest(infohash, s.conf.K)...)
	}

	if len(nodes) == 0 {
		if len(cb) != 0 && cb[0] != nil {
			cb[0](Result{})
		}
		return
	}

	ids := make(metainfo.Hashes, len(nodes))
	for i, node := range nodes {
		ids[i] = node.ID
	}

	for _, node := range nodes {
		s.getPeers(infohash, node.Addr, s.conf.SearchDepth, ids, cb...)
	}

}

// AnnouncePeer announces the torrent infohash to the K closest nodes,
// and returns the nodes to which it sends the announce_peer query.
func (s *Server) AnnouncePeer(infohash metainfo.Hash, port uint16, impliedPort bool) []krpc.Node {
	if infohash.IsZero() {
		panic("the infohash of the torrent is ZERO")
	}

	var nodes []krpc.Node
	if s.ipv4 {
		nodes = s.routingTable4.Closest(infohash, s.conf.K)
	}
	if s.ipv6 {
		nodes = append(nodes, s.routingTable6.Closest(infohash, s.conf.K)...)
	}

	sentNodes := make([]krpc.Node, 0, len(nodes))
	for _, node := range nodes {
		addr := node.Addr.Addr()
		token := s.tokenPeerManager.Get(infohash, addr)
		if token == "" {
			continue
		}

		arg := krpc.QueryArg{ImpliedPort: impliedPort, InfoHash: infohash, Port: port, Token: token}
		t := newTransaction(s, addr, queryMethodAnnouncePeer, arg)
		if err := s.request(t); err != nil {
			s.conf.ErrorLog("fail to send query message to '%s': %s", addr.String(), err)
		} else {
			sentNodes = append(sentNodes, node)
		}
	}

	return sentNodes
}

// FindNode sends the "find_node" query to the addr to find the target node.
//
// Notice: In general, it's used to bootstrap the routing table.
func (s *Server) FindNode(addr net.Addr, target metainfo.Hash) error {
	if target.IsZero() {
		panic("the target is ZERO")
	}

	return s.findNode(target, addr, s.conf.SearchDepth, nil)
}

func (s *Server) findNode(target metainfo.Hash, addr net.Addr, depth int,
	ids metainfo.Hashes) error {
	arg := krpc.QueryArg{Target: target, Wants: s.want}
	t := newTransaction(s, addr, queryMethodFindNode, arg)
	t.OnResponse = s.onFindNodeResp
	t.Visited = ids
	return s.request(t)
}

func (s *Server) onFindNodeResp(t *transaction, a net.Addr, m krpc.Message) {
	// Search the target node recursively.
	t.Depth--
	if t.Depth < 1 {
		return
	}

	var found bool
	ids := t.Visited
	nodes := make([]krpc.Node, 0, len(m.R.Nodes)+len(m.R.Nodes6))
	for _, node := range m.R.Nodes {
		if node.ID == t.Arg.Target {
			found = true
		}
		if ids.Contains(node.ID) {
			continue
		}
		if s.addNode2(node, m.RO) == NodeAdded {
			nodes = append(nodes, node)
			ids = append(ids, node.ID)
		}
	}
	for _, node := range m.R.Nodes6 {
		if node.ID == t.Arg.Target {
			found = true
		}
		if ids.Contains(node.ID) {
			continue
		}
		if s.addNode2(node, m.RO) == NodeAdded {
			nodes = append(nodes, node)
			ids = append(ids, node.ID)
		}
	}

	if found || len(nodes) == 0 {
		return
	}

	for _, node := range nodes {
		err := s.findNode(t.Arg.Target, node.Addr.Addr(), t.Depth, ids)
		if err != nil {
			s.conf.ErrorLog(`fail to send "find_node" query to '%s': %s`,
				node.Addr.String(), err)
		}
	}
}
