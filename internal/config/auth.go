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

package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// AuthConfig is the configuration for the auth package
type AuthConfig struct {
	// TokenKey is the key used to store the provider's token in the database
	TokenKey string `mapstructure:"token_key" default:"./.ssh/token_key_passphrase"`
}

// GetTokenKey returns a key used to encrypt the provider's token in the database
func (acfg *AuthConfig) GetTokenKey() ([]byte, error) {
	return readKey(acfg.TokenKey)
}

func readKey(keypath string) ([]byte, error) {
	cleankeypath := filepath.Clean(keypath)
	data, err := os.ReadFile(cleankeypath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key: %w", err)
	}

	return data, nil
}
