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

package tracker

import (
	"context"
	"sync"

	"github.com/xgfone/bt/metainfo"
)

// GetPeersResult represents the result of getting the peers from the tracker.
type GetPeersResult struct {
	Error   error // nil stands for success. Or, for failure.
	Tracker string
	Resp    AnnounceResponse
}

// GetPeers gets the peers from the trackers.
//
// Notice: the returned chan will be closed when all the requests end.
func GetPeers(ctx context.Context, id, infohash metainfo.Hash, trackers []string) <-chan GetPeersResult {
	conf := ClientConfig{ID: id}
	req := AnnounceRequest{InfoHash: infohash}

	_len := len(trackers)
	clen := _len
	if clen > 10 {
		clen = 10
	}

	clients := make(chan Client, clen)
	resps := make(chan GetPeersResult, _len)

	wg := new(sync.WaitGroup)
	wg.Add(_len)

	go func() {
		wg.Wait()
		close(resps)
	}()

	for i := 0; i < clen; i++ {
		go func(i int) {
			for client := range clients {
				resp, err := getPeers(ctx, wg, client, req)
				resps <- GetPeersResult{
					Tracker: client.String(),
					Error:   err,
					Resp:    resp,
				}
			}
		}(i)
	}

	go func() {
		for _, t := range trackers {
			client, err := NewClient(t, conf)
			if err != nil {
				resps <- GetPeersResult{Error: err}
			} else {
				clients <- client
			}
		}
		close(clients)
	}()

	return resps
}

func getPeers(ctx context.Context, wg *sync.WaitGroup, client Client,
	req AnnounceRequest) (AnnounceResponse, error) {
	defer wg.Done()
	return client.Announce(ctx, req)
}
