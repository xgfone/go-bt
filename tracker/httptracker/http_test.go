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

package httptracker

import (
	"reflect"
	"testing"

	"github.com/xgfone/bt/metainfo"
)

func TestHTTPAnnounceRequest(t *testing.T) {
	infohash := metainfo.NewRandomHash()
	peerid := metainfo.NewRandomHash()
	v1 := AnnounceRequest{
		InfoHash:   infohash,
		PeerID:     peerid,
		Uploaded:   789,
		Downloaded: 456,
		Left:       123,
		Port:       80,
		Event:      123,
		Compact:    true,
	}
	vs := v1.ToQuery()

	var v2 AnnounceRequest
	if err := v2.FromQuery(vs); err != nil {
		t.Fatal(err)
	}

	if v2.InfoHash != infohash {
		t.Error(v2.InfoHash)
	}
	if v2.PeerID != peerid {
		t.Error(v2.PeerID)
	}
	if v2.Uploaded != 789 {
		t.Error(v2.Uploaded)
	}
	if v2.Downloaded != 456 {
		t.Error(v2.Downloaded)
	}
	if v2.Left != 123 {
		t.Error(v2.Left)
	}
	if v2.Port != 80 {
		t.Error(v2.Port)
	}
	if v2.Event != 123 {
		t.Error(v2.Event)
	}
	if !v2.Compact {
		t.Error(v2.Compact)
	}

	if !reflect.DeepEqual(v1, v2) {
		t.Errorf("%v != %v", v1, v2)
	}
}
