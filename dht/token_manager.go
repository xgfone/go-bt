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
	"net"
	"sync"
	"time"

	"github.com/xgfone/bt/utils"
)

// TokenManager is used to manage and validate the token.
//
// TODO: Should we allocate the different token for each node??
type tokenManager struct {
	lock  sync.RWMutex
	last  string
	new   string
	exit  chan struct{}
	addrs sync.Map
}

func newTokenManager() *tokenManager {
	token := utils.RandomString(8)
	return &tokenManager{last: token, new: token, exit: make(chan struct{})}
}

func (tm *tokenManager) updateToken() {
	token := utils.RandomString(8)
	tm.lock.Lock()
	tm.last, tm.new = tm.new, token
	tm.lock.Unlock()
}

func (tm *tokenManager) clear(now time.Time, expired time.Duration) {
	tm.addrs.Range(func(k interface{}, v interface{}) bool {
		if now.Sub(v.(time.Time)) >= expired {
			tm.addrs.Delete(k)
		}
		return true
	})
}

// Start starts the token manager.
func (tm *tokenManager) Start(expired time.Duration) {
	tick1 := time.NewTicker(expired)
	tick2 := time.NewTicker(expired / 2)
	defer tick1.Stop()
	defer tick2.Stop()

	for {
		select {
		case <-tm.exit:
			return
		case <-tick2.C:
			tm.updateToken()
		case now := <-tick1.C:
			tm.clear(now, expired)
		}
	}
}

// Stop stops the token manager.
func (tm *tokenManager) Stop() {
	select {
	case <-tm.exit:
	default:
		close(tm.exit)
	}
}

// Token allocates a token for a node addr and returns the token.
func (tm *tokenManager) Token(addr net.Addr) (token string) {
	addrs := addr.String()
	tm.lock.RLock()
	token = tm.new
	tm.lock.RUnlock()
	tm.addrs.Store(addrs, time.Now())
	return
}

// Check checks whether the token associated with the node addr is valid,
// that's, it's not expired.
func (tm *tokenManager) Check(addr net.Addr, token string) (ok bool) {
	tm.lock.RLock()
	last, new := tm.last, tm.new
	tm.lock.RUnlock()

	if last == token || new == token {
		_, ok = tm.addrs.Load(addr.String())
	}

	return
}
