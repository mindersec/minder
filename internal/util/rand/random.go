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

// Package rand contains utility functions largely for unit testing.
// WARNING: Do not use the functions in this package that generate rand / seeds
// for any security related purposes, outside of testing.
package rand

import (
	"math/rand"
	"net"
	"sync/atomic"
)

// seedGuard makes sure each call to NewRand is initialized with a unique seed.
var seedGuard atomic.Int64

// NewRand returns a new instance of rand.Rand with a fixed source.
func NewRand(seed int64) *rand.Rand {
	seedGuard.Add(1)
	return rand.New(rand.NewSource(seed + seedGuard.Load()))
}

// RandomInt returns a random integer between min and max.
func RandomInt(min, max int64, seed int64) int64 {
	r := NewRand(seed)
	return min + r.Int63n(max-min+1)
}

// RandomString returns a random string of length n.
func RandomString(n int, seed int64) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	s := make([]byte, n)
	r := NewRand(seed)
	for i := range s {
		s[i] = letters[r.Intn(len(letters))]
	}
	return string(s)
}

// RandomName returns a random name.
func RandomName(seed int64) string {
	return RandomString(10, seed)
}

// RandomFrom returns an item chosen at random from the given set of
// alternatives.
func RandomFrom[T any](choices []T, seed int64) T {
	idx := RandomInt(0, int64(len(choices)-1), seed)
	return choices[idx]
}

// GetRandomPort returns a random port number.
// The binding address should not need to be configurable
// as this is a short-lived operation just to discover a random available port.
// Note that there is a possible race condition here if another process binds
// to the same port between the time we discover it and the time we use it.
// This is unlikely to happen in practice, but if it does, the user will
// need to retry the command.
// Marking a nosec here because we want this to listen on all addresses to
// ensure a reliable connection chance for the client. This is based on lessons
// learned from the sigstore CLI.
func GetRandomPort() (int, error) {
	listener, err := net.Listen("tcp", ":0") // #nosec
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	return port, nil
}
