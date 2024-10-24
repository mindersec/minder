// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reminder

import (
	"os"

	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/util"
)

// LoggingConfig is the configuration for the logger
type LoggingConfig struct {
	Level string `mapstructure:"level" default:"info"`
}

// LoggerFromConfigFlags creates a new logger from the provided configuration
func LoggerFromConfigFlags(cfg LoggingConfig) zerolog.Logger {
	level := util.ViperLogLevelToZerologLevel(cfg.Level)
	return zerolog.New(os.Stdout).Level(level).With().Timestamp().Logger()
}
