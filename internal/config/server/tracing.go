// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

// TracingConfig is the configuration for our tracing capabilities
type TracingConfig struct {
	Enabled bool `mapstructure:"enabled" default:"false"`
	// for the demonstration, we use AlwaysSmaple sampler to take all spans.
	// do not use this option in production.
	SampleRatio float64 `mapstructure:"sample_ratio" default:"0.1"`
}
