//
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
