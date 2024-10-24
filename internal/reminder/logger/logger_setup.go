// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package logger provides the configuration for the reminder logger
package logger

import (
	"os"

	"github.com/rs/zerolog"

	config "github.com/mindersec/minder/internal/config/reminder"
	"github.com/mindersec/minder/pkg/util"
)

// FromFlags creates a new logger from the provided configuration
func FromFlags(cfg config.LoggingConfig) zerolog.Logger {
	level := util.ViperLogLevelToZerologLevel(cfg.Level)
	return zerolog.New(os.Stdout).Level(level).With().Timestamp().Logger()
}
