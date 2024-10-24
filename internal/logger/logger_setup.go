// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/mindersec/minder/internal/util"
	config "github.com/mindersec/minder/pkg/config/server"
)

// FromFlags configures logging and returns a logger with settings matching
// the supplied cfg.  It also performs some global initialization, because
// that's how zerolog works.
func FromFlags(cfg config.LoggingConfig) zerolog.Logger {
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
