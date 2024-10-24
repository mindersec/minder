// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

// MetricsConfig is the configuration for the metrics
type MetricsConfig struct {
	Enabled bool `mapstructure:"enabled" default:"true"`
}
