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
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/xgfone/go-bt/metainfo"
)

// Handler is used to handle the incoming peer connection.
type Handler interface {
	// OnHandShake is used to check whether the handshake extension is acceptable.
	OnHandShake(conn *PeerConn) error

	// OnMessage is used to handle the incoming peer message.
	//
	// If requires, it should write the response to the peer.
	OnMessage(conn *PeerConn, msg Message) error

	// OnClose is called when the connection is closed, which may be used
	// to do some cleaning work by the handler.
	OnClose(conn *PeerConn)
}

// Config is used to configure the server.
type Config struct {
	// ExtBits is used to handshake with the client.
	ExtBits ExtensionBits

	// MaxLength is used to limit the maximum number of the message body.
	//
	// 0 represents no limit, and the default is 262144, that's, 256KB.
	MaxLength uint32

	// Timeout is used to control the timeout of the read/write the message.
	//
	// The default is 0, which represents no timeout.
	Timeout time.Duration

	// ErrorLog is used to log the error.
	ErrorLog func(format string, args ...interface{}) // Default: log.Printf

	// HandleMessage is used to handle the incoming message. So you can
	// customize it to add the request queue.
	//
	// The default handler is to forward to pc.HandleMessage(msg, handler).
	HandleMessage func(pc *PeerConn, msg Message, handler Handler) error
}

func (c *Config) set(conf *Config) {
	if conf != nil {
		*c = *conf
	}

	if c.MaxLength == 0 {
		c.MaxLength = 262144 // 256KB
	}
	if c.ErrorLog == nil {
		c.ErrorLog = log.Printf
	}
	if c.HandleMessage == nil {
		c.HandleMessage = func(pc *PeerConn, m Message, h Handler) error {
			return pc.HandleMessage(m, h)
		}
	}
}

// Server is used to implement the peer protocol server.
type Server struct {
	net.Listener

	ID      metainfo.Hash // Required
	Handler Handler       // Required
	Config  Config        // Optional
}

// NewServerByListen returns a new Server by listening on the address.
func NewServerByListen(network, address string, id metainfo.Hash, h Handler, c *Config) (*Server, error) {
	ln, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	return NewServer(ln, id, h, c), nil
}

// NewServer returns a new Server.
func NewServer(ln net.Listener, id metainfo.Hash, h Handler, c *Config) *Server {
	if id.IsZero() {
		panic("the peer node id must not be empty")
	}

	var conf Config
	conf.set(c)
	return &Server{Listener: ln, ID: id, Handler: h, Config: conf}
}

// Run starts the peer protocol server.
func (s *Server) Run() {
	s.Config.set(nil)
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			if !strings.Contains(err.Error(), "closed") {
				s.Config.ErrorLog("fail to accept new connection: %s", err)
			}
			return
		}
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	pc := &PeerConn{
		ID:         s.ID,
		Conn:       conn,
		ExtBits:    s.Config.ExtBits,
		Timeout:    s.Config.Timeout,
		MaxLength:  s.Config.MaxLength,
		Choked:     true,
		PeerChoked: true,
	}

	if err := s.handlePeerMessage(pc); err != nil {
		s.Config.ErrorLog(err.Error())
	}
}

func (s *Server) handlePeerMessage(pc *PeerConn) (err error) {
	defer pc.Close()
	if err = pc.Handshake(); err != nil {
		return fmt.Errorf("fail to handshake with '%s': %s", pc.RemoteAddr().String(), err)
	} else if err = s.Handler.OnHandShake(pc); err != nil {
		return fmt.Errorf("handshake error with '%s': %s", pc.RemoteAddr().String(), err)
	}

	defer s.Handler.OnClose(pc)
	return s.loopRun(pc, s.Handler)
}

// LoopRun loops running Read-Handle message.
func (s *Server) loopRun(pc *PeerConn, handler Handler) error {
	for {
		msg, err := pc.ReadMsg()
		switch err {
		case nil:
		case io.EOF:
			return nil
		default:
			s := err.Error()
			if strings.Contains(s, "closed") {
				return nil
			}
			return fmt.Errorf("fail to decode the message from '%s': %s",
				pc.RemoteAddr().String(), s)
		}

		if err = s.Config.HandleMessage(pc, msg, handler); err != nil {
			return fmt.Errorf("fail to handle peer message from '%s': %s",
				pc.RemoteAddr().String(), err)
		}
	}
}
