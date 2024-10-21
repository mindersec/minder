// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reminder

// LoggingConfig is the configuration for the logger
type LoggingConfig struct {
	Level string `mapstructure:"level" default:"info"`
}
