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
		return "", fmt.Errorf("We are before 1970, invalid nonce timestamp: %d", timestamp)
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
