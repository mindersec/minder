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

// Package keystores contains logic for loading encryption keys from a keystores
package keystores

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	serverconfig "github.com/stacklok/minder/internal/config/server"
)

// KeyStore represents a struct which stores or can fetch encryption keys.
type KeyStore interface {
	// GetKey retrieves the key for the specified algorithm by key ID.
	GetKey(id string) ([]byte, error)
}

// LocalKeyStore is the config value for an on-disk key store
const LocalKeyStore = "local"

// ErrUnknownKeyID is returned when the Key ID cannot be found by the keystore.
var ErrUnknownKeyID = errors.New("unknown key id")

type keysByID map[string][]byte

// NewKeyStoreFromConfig creates an instance of a KeyStore based on the
// AuthConfig in Minder.
// Since our only implementation is based on reading from the local disk, do
// all key loading during construction of the struct.
func NewKeyStoreFromConfig(config serverconfig.CryptoConfig) (KeyStore, error) {
	// TODO: support other methods in future
	if config.KeyStore.Type != LocalKeyStore {
		return nil, fmt.Errorf("unexpected keystore type: %s", config.KeyStore.Type)
	}

	if config.KeyStore.Local.KeyDir == "" {
		return nil, errors.New("key directory not defined in keystore config")
	}

	// Join the default key to the fallback keys to assemble the full
	// set of keys to load.
	keyIDs := []string{config.Default.KeyID}
	fallbackKeyID := ""
	if config.Fallback.KeyID != "" {
		keyIDs = append(keyIDs, config.Fallback.KeyID)
		fallbackKeyID = config.Fallback.KeyID
	}
	keys := make(keysByID, len(keyIDs))
	for _, keyID := range keyIDs {
		key, err := readKey(config.KeyStore.Local.KeyDir, keyID)
		if err != nil {
			return nil, fmt.Errorf("unable to read key %s: %w", keyID, err)
		}
		keys[keyID] = key
	}

	return &localFileKeyStore{
		keys:          keys,
		fallbackKeyID: fallbackKeyID,
	}, nil
}

// NewKeyStoreFromMap constructs a keystore from a map of key ID to key bytes.
// This is mostly useful for testing.
func NewKeyStoreFromMap(keys keysByID, fallbackID string) KeyStore {
	return &localFileKeyStore{keys, fallbackID}
}

type localFileKeyStore struct {
	keys          keysByID
	fallbackKeyID string
}

func (l *localFileKeyStore) GetKey(id string) ([]byte, error) {
	if id == "" {
		if l.fallbackKeyID != "" {
			id = l.fallbackKeyID
		} else {
			return nil, errors.New("empty key ID with no config defined")
		}
	}
	key, ok := l.keys[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownKeyID, id)
	}
	return key, nil
}

func readKey(keyDir string, keyFilename string) ([]byte, error) {
	keyPath := filepath.Join(keyDir, keyFilename)
	cleanPath := filepath.Clean(keyPath)

	// NOTE: Minder reads the base64 encoded key PLUS line feed character and
	// uses it as the key without decoding. The CFB algorithm expects the line
	// feed, and stripping  it out will break existing secrets. The GCM
	// algorithm will base64 decode the key and get rid of the newline.
	//
	// If we get rid of the CFB cipher in future, we should base64 decode the
	// string here and always use the decoded bytes minus linefeed as the key
	// in the algorithms.
	key, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key: %w", err)
	}
	return key, nil
}
