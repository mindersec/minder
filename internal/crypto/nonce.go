// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// GenerateNonce generates a nonce for the OAuth2 flow. The nonce is a base64 encoded
func GenerateNonce() (string, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	nonceBytes := make([]byte, 8)
	timestamp := time.Now().Unix()
	if timestamp < 0 {
		return "", fmt.Errorf("we are before 1970, invalid nonce timestamp: %d", timestamp)
	}
	binary.BigEndian.PutUint64(nonceBytes, uint64(timestamp))

	nonceBytes = append(nonceBytes, randomBytes...)
	nonce := base64.RawURLEncoding.EncodeToString(nonceBytes)
	return nonce, nil
}

// IsNonceValid checks if a nonce is valid. A nonce is valid if it is a base64 encoded string
func IsNonceValid(nonce string, noncePeriod int64) (bool, error) {
	nonceBytes, err := base64.RawURLEncoding.DecodeString(nonce)
	if err != nil {
		return false, err
	}

	if len(nonceBytes) < 8 {
		return false, nil
	}

	storedTimestamp := binary.BigEndian.Uint64(nonceBytes[:8])
	if storedTimestamp > math.MaxInt64 {
		// stored timestamp is before Unix epoch -> bad value
		return false, nil
	}
	currentTimestamp := time.Now().Unix()
	// Checked explicitly for overflow in stored timestamp
	// nolint: gosec
	timeDiff := currentTimestamp - int64(storedTimestamp)

	if timeDiff > noncePeriod { // 5 minutes = 300 seconds
		return false, nil
	}

	return true, nil
}
