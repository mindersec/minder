// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	config "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/util"
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

	logger := zerolog.New(zerolog.MultiLevelWriter(loggers...)).With().Timestamp().Logger()

	// Use this logger when calling zerolog.Ctx(nil), etc
	zerolog.DefaultContextLogger = &logger
	return logger
}
