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
	KeyStore KeyStoreConfig `mapstructure:"key_store"`
	Default  DefaultCrypto  `mapstructure:"default"`
	Fallback FallbackCrypto `mapstructure:"fallback"`
}

// KeyStoreConfig specifies the type of keystore to use and its configuration
type KeyStoreConfig struct {
	Type string `mapstructure:"type" default:"local"`
	// is currently expected to match the structure of LocalFileKeyStoreConfig
	Config map[string]any `mapstructure:"config"`
}

// DefaultCrypto defines the default crypto to be used for new data
type DefaultCrypto struct {
	KeyID     string `mapstructure:"key_id"`
	Algorithm string `mapstructure:"algorithm"`
}

// FallbackCrypto defines the optional list of keys and algorithms to fall
// back to.
// When rotating keys or algorithms, add the old ones here.
type FallbackCrypto struct {
	KeyIDs     []string `mapstructure:"key_ids"`
	Algorithms []string `mapstructure:"algorithms"`
}

// LocalFileKeyStoreConfig contains configuration for the local file keystore
type LocalFileKeyStoreConfig struct {
	KeyDir string `mapstructure:"key_dir"`
}
