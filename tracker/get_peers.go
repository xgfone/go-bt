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
	"net/url"
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
func GetPeers(ctx context.Context, id, infohash metainfo.Hash, trackers []string) []GetPeersResult {
	if len(trackers) == 0 {
		return nil
	}

	for i, t := range trackers {
		if u, err := url.Parse(t); err == nil && u.Path == "" {
			u.Path = "/announce"
			trackers[i] = u.String()
		}
	}

	_len := len(trackers)
	wlen := _len
	if wlen > 10 {
		wlen = 10
	}

	reqs := make(chan string, wlen)
	go func() {
		for i := 0; i < _len; i++ {
			reqs <- trackers[i]
		}
	}()

	wg := new(sync.WaitGroup)
	wg.Add(_len)

	var lock sync.Mutex
	results := make([]GetPeersResult, 0, _len)
	for i := 0; i < wlen; i++ {
		go func() {
			for tracker := range reqs {
				resp, err := getPeers(ctx, wg, tracker, id, infohash)
				lock.Lock()
				results = append(results, GetPeersResult{
					Tracker: tracker,
					Error:   err,
					Resp:    resp,
				})
				lock.Unlock()
			}
		}()
	}

	wg.Wait()
	close(reqs)

	return results
}

func getPeers(ctx context.Context, wg *sync.WaitGroup, tracker string,
	nodeID, infoHash metainfo.Hash) (resp AnnounceResponse, err error) {
	defer wg.Done()

	client, err := NewClient(tracker, ClientConfig{ID: nodeID})
	if err == nil {
		resp, err = client.Announce(ctx, AnnounceRequest{InfoHash: infoHash})
	}

	return
}
