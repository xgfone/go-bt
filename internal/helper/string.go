// Copyright 2023 xgfone
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

package helper

import (
	crand "crypto/rand"
	"math/rand"
)

// RandomString generates a size-length string randomly.
func RandomString(size int) string {
	bs := make([]byte, size)
	if n, _ := crand.Read(bs); n < size {
		for ; n < size; n++ {
			bs[n] = byte(rand.Intn(256))
		}
	}
	return string(bs)
}
