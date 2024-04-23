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
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/config"
)

// Config contains the configuration for the reminder service
type Config struct {
	Database         config.DatabaseConfig `mapstructure:"database"`
	RecurrenceConfig RecurrenceConfig      `mapstructure:"recurrence"`
	EventConfig      EventConfig           `mapstructure:"events"`
	LoggingConfig    LoggingConfig         `mapstructure:"logging"`
	CursorFile       string                `mapstructure:"cursor_file" default:"/tmp/reminder_cursor"`
}

// Normalize normalizes the configuration
// Returns a boolean indicating if the config was modified and an error if the config is invalid
func (c *Config) Normalize(cmd *cobra.Command) (bool, error) {
	err := c.validate()
	if err != nil {
		return c.patchConfig(cmd, err)
	}

	return false, nil
}

func (c *Config) validate() error {
	if c == nil {
		return errors.New("config cannot be nil")
	}

	err := c.RecurrenceConfig.Validate()
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) patchConfig(cmd *cobra.Command, err error) (bool, error) {
	var batchSizeErr *InvalidBatchSizeError
	if errors.As(err, &batchSizeErr) {
		minAllowedBatchSize := batchSizeErr.MaxPerProject * batchSizeErr.MinProjectFetchLimit
		cmd.Println("⚠ WARNING: " + batchSizeErr.Error())
		cmd.Printf("Setting batch size to minimum allowed value: %d\n", minAllowedBatchSize)

		// Update the config with the minimum allowed batch size
		c.RecurrenceConfig.BatchSize = minAllowedBatchSize
		return true, nil
	}

	return false, fmt.Errorf("invalid config: %w", err)
}

// SetViperDefaults sets the default values for the configuration to be picked up by viper
func SetViperDefaults(v *viper.Viper) {
	v.SetEnvPrefix("reminder")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	config.SetViperStructDefaults(v, "", Config{})
}

// RegisterReminderFlags registers the flags for the minder cli
func RegisterReminderFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	viperPath := "cursor_file"
	if err := config.BindConfigFlag(v, flags, viperPath, "cursor-file",
		v.GetString(viperPath), "DB Cursor file path for reminder", flags.String); err != nil {
		return err
	}

	viperPath = "logging.level"
	if err := config.BindConfigFlag(v, flags, viperPath, "logging-level",
		v.GetString(viperPath), "Logging level for reminder", flags.String); err != nil {
		return err
	}

	if err := config.RegisterDatabaseFlags(v, flags); err != nil {
		return err
	}

	return registerRecurrenceFlags(v, flags)
}
