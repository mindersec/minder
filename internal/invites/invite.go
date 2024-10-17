// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package invites provides the invite utilities for minder
package invites

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
