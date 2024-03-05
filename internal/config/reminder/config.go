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

// Package reminder contains configuration options for the reminder service.
package reminder

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/config"
)

// Config contains the configuration for the reminder service
type Config struct {
	Database         config.DatabaseConfig `mapstructure:"database"`
	RecurrenceConfig RecurrenceConfig      `mapstructure:"recurrence"`
	EventConfig      EventConfig           `mapstructure:"events"`
	CursorFile       string                `mapstructure:"cursor_file"`
	LoggingLevel     string                `mapstructure:"logging_level"`
}

// RecurrenceConfig contains the configuration for the reminder recurrence
type RecurrenceConfig struct {
	// Interval is the time between reminders
	Interval string `mapstructure:"interval"`
	// BatchSize is the number of reminders to process at once. Batch size cannot be less than
	// MaxPerProject * MinProjectFetchLimit.
	BatchSize int `mapstructure:"batch_size"`
	// MaxPerProject is the maximum number of reminders per project in a batch
	MaxPerProject int `mapstructure:"max_per_project"`
	// MinProjectFetchLimit is the minimum number of projects to fetch in an iteration. Additional
	// projects are fetched if there is still space in the batch.
	MinProjectFetchLimit int `mapstructure:"min_project_fetch_limit"`
	// MinElapsed is the minimum time after last update before sending a reminder
	MinElapsed string `mapstructure:"min_elapsed"`
}

// ValidateConfig validates the configuration
func ValidateConfig(cfg *Config) error {
	if cfg.RecurrenceConfig.BatchSize <
		cfg.RecurrenceConfig.MaxPerProject*cfg.RecurrenceConfig.MinProjectFetchLimit {
		return fmt.Errorf("batch_size %d cannot be less than max_per_project(%d)*min_project_fetch_limit(%d)=%d",
			cfg.RecurrenceConfig.BatchSize,
			cfg.RecurrenceConfig.MaxPerProject,
			cfg.RecurrenceConfig.MinProjectFetchLimit,
			cfg.RecurrenceConfig.MaxPerProject*cfg.RecurrenceConfig.MinProjectFetchLimit)
	}

	minElapsed := cfg.RecurrenceConfig.MinElapsed
	if _, err := time.ParseDuration(minElapsed); err != nil {
		return fmt.Errorf("invalid min_elapsed: %s", minElapsed)
	}

	interval := cfg.RecurrenceConfig.Interval
	if _, err := time.ParseDuration(interval); err != nil {
		return fmt.Errorf("invalid interval: %s", interval)
	}

	if cfg.EventConfig == (EventConfig{}) {
		return fmt.Errorf("event config is empty, required for sending events to minder server")
	}

	return nil
}

// RegisterReminderFlags registers the flags for the minder cli
func RegisterReminderFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	if err := config.BindConfigFlag(v, flags, "cursor_file", "cursor-file",
		"/tmp/reminder-cursor", "DB Cursor file path for reminder", flags.String); err != nil {
		return err
	}

	if err := config.BindConfigFlag(v, flags, "logging_level", "logging-level",
		"debug", "Logging level for reminder", flags.String); err != nil {
		return err
	}

	if err := config.RegisterDatabaseFlags(v, flags); err != nil {
		return err
	}

	return registerRecurrenceFlags(v, flags)
}

func registerRecurrenceFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	err := config.BindConfigFlagWithShort(
		v, flags, "recurrence.interval", "interval", "i", "1h",
		"Interval between reminders", flags.StringP)
	if err != nil {
		return err
	}

	err = config.BindConfigFlagWithShort(
		v, flags, "recurrence.batch_size", "batch-size", "b", 100,
		"Number of reminders to process at once", flags.IntP)
	if err != nil {
		return err
	}

	err = config.BindConfigFlag(
		v, flags, "recurrence.max_per_project", "max-per-project", 10,
		"Maximum number of reminders per project in a batch", flags.Int)
	if err != nil {
		return err
	}

	err = config.BindConfigFlag(
		v, flags, "recurrence.min_project_fetch_limit", "min-project-fetch-limit", 10,
		"Minimum No. of projects to fetch in an iteration", flags.Int)
	if err != nil {
		return err
	}

	return config.BindConfigFlag(
		v, flags, "recurrence.min_elapsed", "min-elapsed", "1h",
		"Minimum time after last update before sending a reminder", flags.String,
	)
}
