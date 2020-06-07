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

package utils

import (
	"sync/atomic"
)

// Bool is used to implement a atomic bool.
type Bool struct {
	v uint32
}

// NewBool returns a new Bool with the initialized value.
func NewBool(t bool) (b Bool) {
	if t {
		b.v = 1
	}
	return
}

// Get returns the bool value.
func (b *Bool) Get() bool { return atomic.LoadUint32(&b.v) == 1 }

// SetTrue sets the bool value to true.
func (b *Bool) SetTrue() { atomic.StoreUint32(&b.v, 1) }

// SetFalse sets the bool value to false.
func (b *Bool) SetFalse() { atomic.StoreUint32(&b.v, 0) }

// Set sets the bool value to t.
func (b *Bool) Set(t bool) {
	if t {
		b.SetTrue()
	} else {
		b.SetFalse()
	}
}
