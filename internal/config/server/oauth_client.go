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

package server

import "fmt"

type OAuthClientConfig struct {
	ClientID         string `mapstructure:"client_id"`
	ClientIDFile     string `mapstructure:"client_id_file"`
	ClientSecret     string `mapstructure:"client_secret"`
	ClientSecretFile string `mapstructure:"client_secret_file"`
	RedirectURI      string `mapstructure:"redirect_uri"`
}

func (cfg *OAuthClientConfig) GetClientID() (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("OAuthClientConfig is nil")
	}
	return fileOrArg(cfg.ClientIDFile, cfg.ClientID, "client ID")
}

func (cfg *OAuthClientConfig) GetClientSecret() (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("OAuthClientConfig is nil")
	}
	return fileOrArg(cfg.ClientSecretFile, cfg.ClientSecret, "client secret")
}
