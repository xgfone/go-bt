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

package metainfo

import (
	"testing"
)

func TestAddress(t *testing.T) {
	var addr1 Address
	if err := addr1.FromString("1.2.3.4:1234"); err != nil {
		t.Error(err)
		return
	}

	data, err := addr1.MarshalBencode()
	if err != nil {
		t.Error(err)
		return
	} else if s := string(data); s != `l7:1.2.3.4i1234ee` {
		t.Errorf(`expected 'l7:1.2.3.4i1234ee', but got '%s'`, s)
	}

	var addr2 Address
	if err = addr2.UnmarshalBencode(data); err != nil {
		t.Error(err)
	} else if addr2.String() != `1.2.3.4:1234` {
		t.Errorf("expected '1.2.3.4:1234', but got '%s'", addr2)
	}

	if data, err = addr2.MarshalBinary(); err != nil {
		t.Error(err)
	} else if s := string(data); s != "\x01\x02\x03\x04\x04\xd2" {
		t.Errorf(`expected '\x01\x02\x03\x04\x04\xd2', but got '%#x'`, s)
	}
}
