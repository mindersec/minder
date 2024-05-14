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
	"path/filepath"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/crypto/algorithms"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// KeyStore represents a struct which stores or can fetch encryption keys.
type KeyStore interface {
	// GetKey retrieves the key for the specified algorithm by key ID.
	GetKey(algorithm algorithms.Type, id string) ([]byte, error)
}

// ErrUnknownKeyID is returned when the Key ID cannot be found by the keystore.
var ErrUnknownKeyID = errors.New("unknown key id")

// This structure is used by the keystore implementation to manage keys.
type keysByAlgorithmAndID map[algorithms.Type]map[string][]byte

// NewKeyStoreFromConfig creates an instance of a KeyStore based on the
// AuthConfig in Minder.
// Since our only implementation is based on reading from the local disk, do
// all key loading during construction of the struct.
// TODO: allow support for multiple keys/algos
func NewKeyStoreFromConfig(config *serverconfig.AuthConfig) (KeyStore, error) {
	key, err := config.GetTokenKey()
	if err != nil {
		return nil, fmt.Errorf("unable to read encryption key from %s: %w", config.TokenKey, err)
	}
	// Use the key filename as the key ID.
	name := filepath.Base(config.TokenKey)
	keys := map[algorithms.Type]map[string][]byte{
		algorithms.Aes256Cfb: {
			name: key,
		},
	}
	return &localFileKeyStore{
		keys: keys,
	}, nil
}

type localFileKeyStore struct {
	keys keysByAlgorithmAndID
}

func (l *localFileKeyStore) GetKey(algorithm algorithms.Type, id string) ([]byte, error) {
	algorithmKeys, ok := l.keys[algorithm]
	if !ok {
		return nil, fmt.Errorf("%w: %s", algorithms.ErrUnknownAlgorithm, algorithm)
	}
	key, ok := algorithmKeys[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownKeyID, id)
	}
	return key, nil
}
