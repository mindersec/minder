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

// CryptoConfig is the configuration for the crypto package
type CryptoConfig struct {
	Memory      uint32 `mapstructure:"memory" default:"65536"`
	Iterations  uint32 `mapstructure:"iterations" default:"50"`
	Parallelism uint   `mapstructure:"parallelism" default:"4"`
	SaltLength  uint32 `mapstructure:"salt_length" default:"16"`
	KeyLength   uint32 `mapstructure:"key_length" default:"32"`
}

// GetCryptoConfigWithDefaults returns a CryptoConfig with default values
// TODO: extract from struct default tags
func GetCryptoConfigWithDefaults() CryptoConfig {
	return DefaultConfigForTest().Salt
}
