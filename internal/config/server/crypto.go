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

package server

// CryptoConfig is the configuration for the crypto engine
type CryptoConfig struct {
	KeyStore         KeyStoreConfig `mapstructure:"key_store"`
	DefaultKeyID     string         `mapstructure:"default_key_id"`
	DefaultAlgorithm string         `mapstructure:"default_algorithm"`
	// Optional list of keys and algorithms to fall back to.
	// When rotating keys or algorithms, add the old ones here.
	FallbackKeyIDs     []string `mapstructure:"fallback_key_ids"`
	FallbackAlgorithms []string `mapstructure:"fallback_algorithms"`
}

// KeyStoreConfig specifies the type of keystore to use and its configuration
type KeyStoreConfig struct {
	Type string `mapstructure:"type"`
	// is currently expected to match the structure of LocalFileKeyStoreConfig
	Config map[string]any `mapstructure:"config"`
}

// LocalFileKeyStoreConfig contains configuration for the local file keystore
type LocalFileKeyStoreConfig struct {
	KeyDir string `mapstructure:"key_dir"`
}
