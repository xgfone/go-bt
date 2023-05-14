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

package httptracker

import (
	"reflect"
	"testing"
)

func TestPeers(t *testing.T) {
	peers := Peers{
		{ID: "123", IP: "1.1.1.1", Port: 80},
		{ID: "456", IP: "2.2.2.2", Port: 81},
	}

	b, err := peers.MarshalBencode()
	if err != nil {
		t.Fatal(err)
	}

	var ps Peers
	if err = ps.UnmarshalBencode(b); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(ps, peers) {
		t.Errorf("%v != %v", ps, peers)
	}

	/// For BEP 23
	peers = Peers{
		{IP: "1.1.1.1", Port: 80},
		{IP: "2.2.2.2", Port: 81},
	}

	b, err = peers.MarshalBencode()
	if err != nil {
		t.Fatal(err)
	}

	if err = ps.UnmarshalBencode(b); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(ps, peers) {
		t.Errorf("%v != %v", ps, peers)
	}
}

func TestPeers6(t *testing.T) {
	peers := Peers6{
		{IP: "fe80::5054:ff:fef0:1ab", Port: 80},
		{IP: "fe80::5054:ff:fe29:205d", Port: 81},
	}

	b, err := peers.MarshalBencode()
	if err != nil {
		t.Fatal(err)
	}

	var ps Peers6
	if err = ps.UnmarshalBencode(b); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(ps, peers) {
		t.Errorf("%v != %v", ps, peers)
	}
}
