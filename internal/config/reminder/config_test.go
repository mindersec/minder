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

package reminder_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stacklok/minder/internal/config"
	"github.com/stacklok/minder/internal/config/reminder"
)

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		config           reminder.Config
		normalizedConfig reminder.Config
		modified         bool
		errMsg           string
	}{
		{
			name: "ValidValues",
			config: reminder.Config{
				RecurrenceConfig: reminder.RecurrenceConfig{
					Interval:             parseTimeDuration(t, "1h"),
					BatchSize:            100,
					MaxPerProject:        10,
					MinProjectFetchLimit: 5,
					MinElapsed:           parseTimeDuration(t, "1h"),
				},
				EventConfig: reminder.EventConfig{
					Connection: config.DatabaseConfig{
						Port: 8080,
					},
				},
			},
		},
		{
			name: "InvalidBatchSize",
			config: reminder.Config{
				RecurrenceConfig: reminder.RecurrenceConfig{
					Interval:             parseTimeDuration(t, "1h"),
					BatchSize:            10,
					MaxPerProject:        10,
					MinProjectFetchLimit: 5,
					MinElapsed:           parseTimeDuration(t, "1h"),
				},
			},
			normalizedConfig: reminder.Config{
				RecurrenceConfig: reminder.RecurrenceConfig{
					Interval:             parseTimeDuration(t, "1h"),
					BatchSize:            50,
					MaxPerProject:        10,
					MinProjectFetchLimit: 5,
					MinElapsed:           parseTimeDuration(t, "1h"),
				},
			},
			modified: true,
		},
		{
			name: "NegativeInterval",
			config: reminder.Config{
				RecurrenceConfig: reminder.RecurrenceConfig{
					Interval:             parseTimeDuration(t, "-1h"),
					BatchSize:            100,
					MaxPerProject:        10,
					MinProjectFetchLimit: 5,
					MinElapsed:           parseTimeDuration(t, "1h"),
				},
			},
			errMsg: "cannot be negative",
		},
		{
			name: "NegativeMinElapsed",
			config: reminder.Config{
				RecurrenceConfig: reminder.RecurrenceConfig{
					Interval:             parseTimeDuration(t, "1h"),
					BatchSize:            100,
					MaxPerProject:        10,
					MinProjectFetchLimit: 5,
					MinElapsed:           parseTimeDuration(t, "-1h"),
				},
			},
			errMsg: "cannot be negative",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			modified, err := tt.config.Normalize(&cobra.Command{})
			if tt.errMsg != "" {
				assert.ErrorContains(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
				if tt.modified {
					assert.True(t, modified)
					assert.Equal(t, tt.normalizedConfig, tt.config)
				}
			}
		})
	}
}

func TestReadConfig(t *testing.T) {
	t.Parallel()

	cfgstr := `---
recurrence:
  interval: "1m"
  batch_size: 100
  max_per_project: 10
  min_project_fetch_limit: 10
  min_elapsed: "1h"
`

	cfgbuf := bytes.NewBufferString(cfgstr)

	v := viper.New()
	reminder.SetViperDefaults(v)

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(cfgbuf), "Unexpected error")

	cfg, err := config.ReadConfigFromViper[reminder.Config](v)
	require.NoError(t, err, "Unexpected error")

	require.Equal(t, parseTimeDuration(t, "1m"), cfg.RecurrenceConfig.Interval)
	require.Equal(t, 100, cfg.RecurrenceConfig.BatchSize)
	require.Equal(t, 10, cfg.RecurrenceConfig.MaxPerProject)
	require.Equal(t, 10, cfg.RecurrenceConfig.MinProjectFetchLimit)
	require.Equal(t, parseTimeDuration(t, "1h"), cfg.RecurrenceConfig.MinElapsed)
	require.Equal(t, "info", cfg.LoggingConfig.Level)
}

func TestReadConfigWithCommandLineArgOverrides(t *testing.T) {
	t.Parallel()

	cfgstr := `---
recurrence:
  interval: "1m"
  batch_size: 100
  max_per_project: 10
  min_project_fetch_limit: 10
  min_elapsed: "1h"
logging:
  level: "debug"
`

	cfgbuf := bytes.NewBufferString(cfgstr)

	v := viper.New()
	reminder.SetViperDefaults(v)

	flags := pflag.NewFlagSet("test", pflag.ExitOnError)

	require.NoError(t, reminder.RegisterReminderFlags(v, flags), "Unexpected error")

	require.NoError(t, flags.Parse([]string{"--interval=1h", "--batch-size=200", "--max-per-project=20", "--min-project-fetch-limit=20", "--min-elapsed=2h"}))

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(cfgbuf), "Unexpected error")

	cfg, err := config.ReadConfigFromViper[reminder.Config](v)
	require.NoError(t, err, "Unexpected error")

	require.Equal(t, parseTimeDuration(t, "1h"), cfg.RecurrenceConfig.Interval)
	require.Equal(t, 200, cfg.RecurrenceConfig.BatchSize)
	require.Equal(t, 20, cfg.RecurrenceConfig.MaxPerProject)
	require.Equal(t, 20, cfg.RecurrenceConfig.MinProjectFetchLimit)
	require.Equal(t, parseTimeDuration(t, "2h"), cfg.RecurrenceConfig.MinElapsed)
	require.Equal(t, "debug", cfg.LoggingConfig.Level)
}

func TestSetViperDefaults(t *testing.T) {
	t.Parallel()

	v := viper.New()
	reminder.SetViperDefaults(v)

	require.Equal(t, "reminder", v.GetEnvPrefix())
	require.Equal(t, parseTimeDuration(t, "1h"), parseTimeDuration(t, v.GetString("recurrence.interval")))
	require.Equal(t, 100, v.GetInt("recurrence.batch_size"))
	require.Equal(t, 10, v.GetInt("recurrence.max_per_project"))
	require.Equal(t, 10, v.GetInt("recurrence.min_project_fetch_limit"))
	require.Equal(t, parseTimeDuration(t, "1h"), parseTimeDuration(t, v.GetString("recurrence.min_elapsed")))
	require.Equal(t, "reminder", v.GetString("events.sql_connection.dbname"))
	require.Equal(t, "reminder-event-postgres", v.GetString("events.sql_connection.dbhost"))
	require.Equal(t, "reminder-event-postgres", v.GetString("events.sql_connection.dbhost"))
	require.Equal(t, "postgres", v.GetString("events.sql_connection.dbuser"))
}

// TestOverrideConfigByEnvVar tests that the configuration can be overridden by environment variables
// This test is not parallel because it modifies the environment variables which other tests can read
//
// nolint: paralleltest
func TestOverrideConfigByEnvVar(t *testing.T) {
	cfgstr := `---
recurrence:
  interval: "1m"
  batch_size: 100
  max_per_project: 10
  min_project_fetch_limit: 10
  min_elapsed: "1h"
database:
  dbhost: "minder"
`

	cfgbuf := bytes.NewBufferString(cfgstr)

	v := viper.New()
	reminder.SetViperDefaults(v)

	oldValInterval := os.Getenv("REMINDER_RECURRENCE_INTERVAL")
	err := os.Setenv("REMINDER_RECURRENCE_INTERVAL", "1h")
	require.NoError(t, err, "Unexpected error")

	oldValDBHost := os.Getenv("REMINDER_DATABASE_DBHOST")
	err = os.Setenv("REMINDER_DATABASE_DBHOST", "foobar")
	require.NoError(t, err, "Unexpected error")

	v.AutomaticEnv()

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(cfgbuf), "Unexpected error")

	cfg, err := config.ReadConfigFromViper[reminder.Config](v)
	require.NoError(t, err, "Unexpected error")

	if oldValInterval == "" {
		err = os.Unsetenv("REMINDER_RECURRENCE_INTERVAL")
	} else {
		err = os.Setenv("REMINDER_RECURRENCE_INTERVAL", oldValInterval)
	}
	require.NoError(t, err, "Unexpected error")

	if oldValDBHost == "" {
		err = os.Unsetenv("REMINDER_DATABASE_DBHOST")
	} else {
		err = os.Setenv("REMINDER_DATABASE_DBHOST", oldValDBHost)
	}
	require.NoError(t, err, "Unexpected error")

	require.Equal(t, parseTimeDuration(t, "1h"), cfg.RecurrenceConfig.Interval)
	require.Equal(t, 100, cfg.RecurrenceConfig.BatchSize)
	require.Equal(t, 10, cfg.RecurrenceConfig.MaxPerProject)
	require.Equal(t, 10, cfg.RecurrenceConfig.MinProjectFetchLimit)
	require.Equal(t, parseTimeDuration(t, "1h"), cfg.RecurrenceConfig.MinElapsed)
	require.Equal(t, "foobar", cfg.Database.Host)
}

func parseTimeDuration(t *testing.T, duration string) time.Duration {
	t.Helper()

	d, _ := time.ParseDuration(duration)
	return d
}
