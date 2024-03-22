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

import (
	"github.com/stacklok/minder/internal/config"
)

// GitHubAppConfig is the configuration for the GitHub App providers
type GitHubAppConfig struct {
	// AppName is the name of the GitHub App
	AppName string `mapstructure:"app_name"`
	// AppID is the ID of the GitHub App
	AppID int64 `mapstructure:"app_id"`
	// UserID is the ID of the GitHub App user
	UserID int64 `mapstructure:"user_id"`
	// PrivateKey is the GitHub App's private key
	PrivateKey string `mapstructure:"private_key"`
}

// GetPrivateKey returns the GitHub App's private key
func (acfg *GitHubAppConfig) GetPrivateKey() ([]byte, error) {
	return config.ReadKey(acfg.PrivateKey)
}
