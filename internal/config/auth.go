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
	// AccessTokenPrivateKey is the private key used to sign the access token for authn/z
	AccessTokenPrivateKey string `mapstructure:"access_token_private_key" default:"./.ssh/access_token_rsa"`
	// AccessTokenPublicKey is the public key used to verify the access token for authn/z
	AccessTokenPublicKey string `mapstructure:"access_token_public_key" default:"./.ssh/access_token_rsa.pub"`
	// RefreshTokenPrivateKey is the private key used to sign the refresh token for authn/z
	RefreshTokenPrivateKey string `mapstructure:"refresh_token_private_key" default:"./.ssh/refresh_token_rsa"`
	// RefreshTokenPublicKey is the public key used to verify the refresh token for authn/z
	RefreshTokenPublicKey string `mapstructure:"refresh_token_public_key" default:"./.ssh/refresh_token_rsa.pub"`
	// TokenExpiry is the expiry time for the access token in seconds
	TokenExpiry int64 `mapstructure:"token_expiry" default:"3600"`
	// RefreshExpiry is the expiry time for the refresh token in seconds
	RefreshExpiry int64 `mapstructure:"refresh_expiry" default:"86400"`
	// NoncePeriod is the period in seconds for which a nonce is valid
	NoncePeriod int64 `mapstructure:"nonce_period" default:"3600"`
	// TokenKey is the key used to store the provider's token in the database
	TokenKey string `mapstructure:"token_key" default:"./.ssh/token_key_passphrase"`
}

// GetAuthConfigWithDefaults returns a AuthConfig with default values
func GetAuthConfigWithDefaults() AuthConfig {
	return AuthConfig{}
}

// GetAccessTokenPrivateKey returns the private key used to sign the access token
func (acfg *AuthConfig) GetAccessTokenPrivateKey() ([]byte, error) {
	return readKey(acfg.AccessTokenPrivateKey)
}

// GetAccessTokenPublicKey returns the public key used to verify the access token
func (acfg *AuthConfig) GetAccessTokenPublicKey() ([]byte, error) {
	return readKey(acfg.AccessTokenPublicKey)
}

// GetRefreshTokenPrivateKey returns the private key used to sign the refresh token
func (acfg *AuthConfig) GetRefreshTokenPrivateKey() ([]byte, error) {
	return readKey(acfg.RefreshTokenPrivateKey)
}

// GetRefreshTokenPublicKey returns the public key used to verify the refresh token
func (acfg *AuthConfig) GetRefreshTokenPublicKey() ([]byte, error) {
	return readKey(acfg.RefreshTokenPublicKey)
}

func (acfg *AuthConfig) GetTokenKey() ([]byte, error) {
	return readKey(acfg.TokenKey)
}

func readKey(keypath string) ([]byte, error) {
	cleankeypath := filepath.Clean(keypath)
	data, err := os.ReadFile(cleankeypath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key: %s", err)
	}

	return data, nil
}
