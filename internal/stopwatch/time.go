// Copyright 2026 Copyright (c) 2026 Digital Asset (Switzerland) GmbH and/or its affiliates. All rights reserved.
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

package stopwatch

import (
	"time"
)

// Time runs a function and returns how long it took to run.
func Time[T any](fn func() T) (ret T, duration time.Duration) {
	now := time.Now()
	ret = fn()
	duration = time.Now().Sub(now)
	return
}

// TimeU runs a function and returns how long it took to run.
func TimeU(fn func()) (duration time.Duration) {
	now := time.Now()
	fn()
	duration = time.Now().Sub(now)
	return
}

// TimeE runs a function and returns how long it took to run.
func TimeE[T, E any](fn func() (T, E)) (ret1 T, duration time.Duration, ret2 E) {
	now := time.Now()
	ret1, ret2 = fn()
	duration = time.Now().Sub(now)
	return
}
