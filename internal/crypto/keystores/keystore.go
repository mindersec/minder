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

	"github.com/go-viper/mapstructure/v2"

	serverconfig "github.com/stacklok/minder/internal/config/server"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

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
	// You might think that a nil value cannot be deserialized to a struct,
	// yet, here we are.
	if config.KeyStore.Config == nil {
		return nil, errors.New("keystore config is missing")
	}

	// TODO: support other methods in future
	if config.KeyStore.Type != LocalKeyStore {
		return nil, fmt.Errorf("unexpected keystore type: %s", config.KeyStore.Type)
	}

	var keystoreCfg serverconfig.LocalFileKeyStoreConfig
	err := mapstructure.Decode(config.KeyStore.Config, &keystoreCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to read keystore config: %w", err)
	}

	// Join the default key to the fallback keys to assemble the full
	// set of keys to load.
	keyIDs := append([]string{config.Default.KeyID}, config.Fallback.KeyIDs...)
	keys := make(keysByID, len(keyIDs))
	for _, keyID := range keyIDs {
		key, err := readKey(keystoreCfg.KeyDir, keyID)
		if err != nil {
			return nil, fmt.Errorf("unable to read key %s: %w", keyID, err)
		}
		keys[keyID] = key
	}

	return &localFileKeyStore{
		keys: keys,
	}, nil
}

// NewKeyStoreFromMap constructs a keystore from a map of key ID to key bytes.
// This is mostly useful for testing.
func NewKeyStoreFromMap(keys keysByID) KeyStore {
	return &localFileKeyStore{keys}
}

type localFileKeyStore struct {
	keys keysByID
}

func (l *localFileKeyStore) GetKey(id string) ([]byte, error) {
	key, ok := l.keys[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownKeyID, id)
	}
	return key, nil
}

func readKey(keyDir string, keyFilename string) ([]byte, error) {
	keyPath := filepath.Join(keyDir, keyFilename)
	cleanPath := filepath.Clean(keyPath)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key: %w", err)
	}

	return data, nil
}
