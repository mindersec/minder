// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package client

// IdentityConfigWrapper is the configuration wrapper for the identity provider used by minder-cli
type IdentityConfigWrapper struct {
	CLI IdentityConfig `mapstructure:"cli" yaml:"cli" json:"cli"`
}

// IdentityConfig is the configuration for the identity provider used by minder-cli
type IdentityConfig struct {
	// IssuerUrl is the base URL where the identity server is running
	IssuerUrl string `mapstructure:"issuer_url" default:"https://auth.stacklok.com" yaml:"issuer_url" json:"issuer_url"`

	// ClientId is the client ID that identifies the server client ID
	ClientId string `mapstructure:"client_id" default:"minder-cli" yaml:"client_id" json:"client_id"`
}
