// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
