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

package utils

import "io"

// CopyNBuffer is the same as io.CopyN, but uses the given buf as the buffer.
//
// If buf is nil or empty, it will make a new one with 2048.
func CopyNBuffer(dst io.Writer, src io.Reader, n int64, buf []byte) (written int64, err error) {
	if len(buf) == 0 {
		buf = make([]byte, 2048)
	}

	written, err = io.CopyBuffer(dst, io.LimitReader(src, n), buf)
	if written == n {
		return n, nil
	} else if written < n && err == nil {
		// src stopped early; must have been EOF.
		err = io.EOF
	}

	return
}
