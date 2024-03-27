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
	"errors"
	"fmt"
)

// ConfigBundleSource is an enum of valid config sources
type ConfigBundleSource string

const (
	// TgzSource represents a bundle in a .tar.gz file
	TgzSource ConfigBundleSource = "tgz"
	// Unknown is a default value
	Unknown = "unknown"
)

var (
	// ErrInvalidBundleSource indicates the config has an invalid source type
	ErrInvalidBundleSource = errors.New("unexpected bundle source")
)

// MarketplaceConfig holds the config for the marketplace functionality.
type MarketplaceConfig struct {
	Enabled bool                 `mapstructure:"enabled" default:"false"`
	Sources []BundleSourceConfig `mapstructure:"sources"`
}

// BundleSourceConfig holds details about where the bundle gets loaded from
type BundleSourceConfig struct {
	Type     string `mapstructure:"type"`
	Location string `mapstructure:"location"`
}

// GetType returns the source as an enum type, or error if invalid
// TODO: investigate whether mapstructure would allow us to validate during
// deserialization.
func (b *BundleSourceConfig) GetType() (ConfigBundleSource, error) {
	if b.Type == string(TgzSource) {
		return TgzSource, nil
	}
	return Unknown, fmt.Errorf("%w: %s", ErrInvalidBundleSource, b.Type)
}
