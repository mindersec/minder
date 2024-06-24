//
// Copyright 2024 Stacklok, Inc.
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

// Package invite provides the invite utilities for minder
package invite

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// codeByteLength = 48 is the length of the invite code in bytes
	codeByteLength = 48
	// expireIn7Days is the duration for which an invitation is valid - 7 days
	expireIn7Days = 7 * 24 * time.Hour
)

// GenerateCode generates a random invite code
func GenerateCode() string {
	bytes := make([]byte, codeByteLength)
	// Generate random bytes
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// GetExpireIn7Days returns the expiration date of the invitation 7 days from t.Now()
func GetExpireIn7Days(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t.Add(expireIn7Days))
}

// IsExpired checks if the invitation has expired
func IsExpired(t time.Time) bool {
	return time.Now().After(t.Add(expireIn7Days))
}
