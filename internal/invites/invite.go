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
	// expireIn7Days is the default duration for which an invitation is valid - 7 days
	expireIn7Days = 7 * 24 * time.Hour
)

// ComputeExpiration computes the expiration time given a requested duration and server maximum duration.
// If requested is 0 or greater than maxDuration, maxDuration is used.
// If maxDuration is 0 or negative, it defaults to 7 days.
func ComputeExpiration(requested time.Duration, maxDuration time.Duration) time.Time {
	if maxDuration <= 0 {
		maxDuration = expireIn7Days
	}
	duration := requested
	if duration <= 0 || duration > maxDuration {
		duration = maxDuration
	}
	return time.Now().Add(duration)
}

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

// IsInviteExpired checks if an invitation is expired using its expiresAt timestamp.
// Note: In the database schema, expires_at is NOT NULL; the fallback to 7 days
// from updatedAt when expiresAt is zero is retained for backwards compatibility
// with in-memory fake services and test mocks.
func IsInviteExpired(expiresAt time.Time, updatedAt time.Time) bool {
	if !expiresAt.IsZero() {
		return time.Now().After(expiresAt)
	}
	return IsExpired(updatedAt)
}

// GetInviteExpiration returns the expiration timestamp of an invitation.
// Note: In the database schema, expires_at is NOT NULL; the fallback to 7 days
// from updatedAt when expiresAt is zero is retained for backwards compatibility
// with in-memory fake services and test mocks.
func GetInviteExpiration(expiresAt time.Time, updatedAt time.Time) *timestamppb.Timestamp {
	if !expiresAt.IsZero() {
		return timestamppb.New(expiresAt)
	}
	return GetExpireIn7Days(updatedAt)
}
