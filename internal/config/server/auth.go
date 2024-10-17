// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

// AuthConfig is the configuration for the auth package
type AuthConfig struct {
	// NoncePeriod is the period in seconds for which a nonce is valid
	NoncePeriod int64 `mapstructure:"nonce_period" default:"3600"`
	// TokenKey is the key used to store the provider's token in the database
	TokenKey string `mapstructure:"token_key" default:"./.ssh/token_key_passphrase"`
}
