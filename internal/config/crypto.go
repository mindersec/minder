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
	"github.com/spf13/viper"
)

const (
	saltMemory     = 64 * 1024
	saltIterations = 3
	saltParameters = 2
	saltLength     = 16
	saltKeyLength  = 32
)

// CryptoConfig is the configuration for the crypto package
type CryptoConfig struct {
	Memory      uint32 `mapstructure:"memory"`
	Iterations  uint32 `mapstructure:"iterations"`
	Parallelism uint   `mapstructure:"parallelism"`
	SaltLength  uint32 `mapstructure:"salt_length"`
	KeyLength   uint32 `mapstructure:"key_length"`
}

// SetCryptoViperDefaults sets the default values for the crypto configuration
// to be picked up by viper
func SetCryptoViperDefaults(v *viper.Viper) {
	// set default values when not set
	v.SetDefault("salt.memory", saltMemory)
	v.SetDefault("salt.iterations", saltIterations)
	v.SetDefault("salt.parallelism", saltParameters)
	v.SetDefault("salt.salt_length", saltLength)
	v.SetDefault("salt.key_length", saltKeyLength)
}

// GetCryptoConfigWithDefaults returns a CryptoConfig with default values
func GetCryptoConfigWithDefaults() CryptoConfig {
	return CryptoConfig{
		Memory:      saltMemory,
		Iterations:  saltIterations,
		Parallelism: saltParameters,
		SaltLength:  saltLength,
		KeyLength:   saltKeyLength,
	}
}
