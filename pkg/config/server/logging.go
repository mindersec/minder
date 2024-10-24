// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/mindersec/minder/internal/util"
)

// Text is the constant for the text format
const Text = "text"

// LoggingConfig is the configuration for the logging package
type LoggingConfig struct {
	Level   string `mapstructure:"level" default:"debug"`
	Format  string `mapstructure:"format" default:"json"`
	LogFile string `mapstructure:"logFile" default:""`

	// LogPayloads controls whether message payloads are ever logged.
	// For debugging purposes, it may be useful to log the payloads that result
	// in error conditions, but could also leak PII.
	LogPayloads bool `mapstructure:"logPayloads" default:"false"`
}

// LoggerFromConfigFlags configures logging and returns a logger with settings matching
// the supplied cfg. It also performs some global initialization, because
// that's how zerolog works.
func LoggerFromConfigFlags(cfg LoggingConfig) zerolog.Logger {
	zlevel := util.ViperLogLevelToZerologLevel(cfg.Level)
	zerolog.SetGlobalLevel(zlevel)

	loggers := []io.Writer{}

	// Conform to https://github.com/open-telemetry/oteps/blob/main/text/logs/0097-log-data-model.md#example-log-records
	// Unfortunately, these can't be set on a per-logger basis except by ConsoleWriter
	zerolog.ErrorFieldName = "exception.message"
	zerolog.TimestampFieldName = "Timestamp"
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano

	if cfg.LogFile != "" {
		cfg.LogFile = filepath.Clean(cfg.LogFile)
		file, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		// NOTE: we are leaking the open file here
		if err != nil {
			log.Err(err).Msg("Failed to open log file, defaulting to stdout")
		} else {
			loggers = append(loggers, file)
		}
	}

	if cfg.Format == Text {
		loggers = append(loggers, zerolog.NewConsoleWriter())
	} else {
		loggers = append(loggers, os.Stdout)
	}

	logger := zerolog.New(zerolog.MultiLevelWriter(loggers...)).With().
		Caller().
		Timestamp().
		Logger()

	// Use this logger when calling zerolog.Ctx(nil), etc
	zerolog.DefaultContextLogger = &logger
	log.Logger = logger

	return logger
}
