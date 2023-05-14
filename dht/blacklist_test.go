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
	"testing"
	"time"
)

func (bl *blacklist) portsLen() (n int) {
	bl.lock.RLock()
	for _, wp := range bl.ips {
		n += len(wp.Ports)
	}
	bl.lock.RUnlock()
	return
}

func (bl *blacklist) getIPs() []string {
	bl.lock.RLock()
	ips := make([]string, 0, len(bl.ips))
	for ip := range bl.ips {
		ips = append(ips, ip)
	}
	bl.lock.RUnlock()
	return ips
}

func TestMemoryBlacklist(t *testing.T) {
	bl := NewMemoryBlacklist(3, time.Second).(*blacklist)
	defer bl.Close()

	bl.Add("1.1.1.1", 123)
	bl.Add("1.1.1.1", 456)
	bl.Add("1.1.1.1", 789)
	bl.Add("2.2.2.2", 111)
	bl.Add("3.3.3.3", 0)
	bl.Add("4.4.4.4", 222)

	ips := bl.getIPs()
	if len(ips) != 3 {
		t.Error(ips)
	} else {
		for _, ip := range ips {
			switch ip {
			case "1.1.1.1", "2.2.2.2", "3.3.3.3":
			default:
				t.Error(ip)
			}
		}
	}

	if n := bl.portsLen(); n != 4 {
		t.Errorf("expect port num 4, but got %d", n)
	}

	if bl.In("1.1.1.1", 111) || !bl.In("1.1.1.1", 123) {
		t.Fail()
	}
	if !bl.In("3.3.3.3", 111) || bl.In("4.4.4.4", 222) {
		t.Fail()
	}

	bl.Del("3.3.3.3", 0)
	if bl.In("3.3.3.3", 111) {
		t.Fail()
	}
}
