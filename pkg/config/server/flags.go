// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

// FlagsConfig contains the configuration for feature flags
type FlagsConfig struct {
	AppName string `mapstructure:"app_name" default:"minder"`

	GoFeature GoFeatureConfig `mapstructure:"go_feature"`
}

// GoFeatureConfig contains the configuration for the GoFeatureFlag (https://gofeatureflag.org/) provider.
type GoFeatureConfig struct {
	FilePath string `mapstructure:"file_path" default:""`
}
