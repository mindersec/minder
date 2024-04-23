// Copyright 2024 Stacklok, Inc
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

// Package logger provides the configuration for the reminder logger
package logger

import (
	"os"

	"github.com/rs/zerolog"

	config "github.com/stacklok/minder/internal/config/reminder"
	"github.com/stacklok/minder/internal/util"
)

// FromFlags creates a new logger from the provided configuration
func FromFlags(cfg config.LoggingConfig) zerolog.Logger {
	level := util.ViperLogLevelToZerologLevel(cfg.Level)
	return zerolog.New(os.Stdout).Level(level).With().Timestamp().Logger()
}
