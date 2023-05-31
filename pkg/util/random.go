//
// Copyright 2023 Stacklok, Inc.
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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package util

import (
	"math/rand"
)

// NewRand returns a new instance of rand.Rand with a fixed source.
func NewRand(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

// RandomInt returns a random integer between min and max.
func RandomInt(min, max int64, seed int64) int64 {
	r := NewRand(seed)
	return min + r.Int63n(max-min+1)
}

// RandomString returns a random string of length n.
func RandomString(n int, seed int64) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	s := make([]byte, n)
	r := NewRand(seed)
	for i := range s {
		s[i] = letters[r.Intn(len(letters))]
	}
	return string(s)
}

// RandomEmail returns a random email address.
func RandomEmail(seed int64) string {
	return RandomString(10, seed) + "@example.com"
}

// RandomName returns a random name.
func RandomName(seed int64) string {
	return RandomString(10, seed)
}

// RandomPassword returns a random password of length n.
func RandomPassword(n int, seed int64) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	s := make([]byte, n-1)
	r := NewRand(seed)
	for i := range s {
		s[i] = letters[r.Intn(len(letters))]
	}

	// add special chars
	const chars = "!@#?*"
	s[len(s)-1] = chars[r.Intn(len(chars))]

	return string(s)
}
