// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

// LoggingConfig is the configuration for the logging package
type LoggingConfig struct {
	Level   string `mapstructure:"level" default:"debug"`
	Format  string `mapstructure:"format" default:"json"`
	LogFile string `mapstructure:"logFile" default:""`

	// LogPayloads controls whether or not message payloads are ever logged.
	// For debugging purposes, it may be useful to log the payloads that result
	// in error conditions, but could also leak PII.
	LogPayloads bool `mapstructure:"logPayloads" default:"false"`
}
