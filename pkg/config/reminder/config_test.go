// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reminder_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/pkg/config"
	"github.com/mindersec/minder/pkg/config/reminder"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer/constants"
)

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config reminder.Config
		errMsg string
	}{
		{
			name: "ValidValues",
			config: reminder.Config{
				RecurrenceConfig: reminder.RecurrenceConfig{
					Interval:   parseTimeDuration(t, "1h"),
					BatchSize:  100,
					MinElapsed: parseTimeDuration(t, "1h"),
				},
				EventConfig: serverconfig.EventConfig{
					Driver: constants.SQLDriver,
					SQLPubSub: serverconfig.SQLEventConfig{
						Connection: config.DatabaseConfig{
							Port: 8080,
						},
					},
				},
			},
		},
		{
			name: "NegativeInterval",
			config: reminder.Config{
				RecurrenceConfig: reminder.RecurrenceConfig{
					Interval:   parseTimeDuration(t, "-1h"),
					BatchSize:  100,
					MinElapsed: parseTimeDuration(t, "1h"),
				},
				EventConfig: serverconfig.EventConfig{
					Driver: constants.SQLDriver,
				},
			},
			errMsg: "cannot be negative",
		},
		{
			name: "NegativeMinElapsed",
			config: reminder.Config{
				RecurrenceConfig: reminder.RecurrenceConfig{
					Interval:   parseTimeDuration(t, "1h"),
					BatchSize:  100,
					MinElapsed: parseTimeDuration(t, "-1h"),
				},
				EventConfig: serverconfig.EventConfig{
					Driver: constants.SQLDriver,
				},
			},
			errMsg: "cannot be negative",
		},
		{
			name: "UnsupportedDriver",
			config: reminder.Config{
				RecurrenceConfig: reminder.RecurrenceConfig{
					Interval:   parseTimeDuration(t, "1h"),
					BatchSize:  100,
					MinElapsed: parseTimeDuration(t, "1h"),
				},
				EventConfig: serverconfig.EventConfig{
					Driver: constants.GoChannelDriver,
				},
			},
			errMsg: fmt.Sprintf("%s is not supported", constants.GoChannelDriver),
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if tt.errMsg != "" {
				assert.ErrorContains(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
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
	require.Equal(t, parseTimeDuration(t, "1h"), cfg.RecurrenceConfig.MinElapsed)
	require.Equal(t, "info", cfg.LoggingConfig.Level)
}

func TestReadConfigWithCommandLineArgOverrides(t *testing.T) {
	t.Parallel()

	cfgstr := `---
recurrence:
  interval: "1m"
  batch_size: 100
  min_elapsed: "1h"
logging:
  level: "debug"
`

	cfgbuf := bytes.NewBufferString(cfgstr)

	v := viper.New()
	reminder.SetViperDefaults(v)

	flags := pflag.NewFlagSet("test", pflag.ExitOnError)

	require.NoError(t, reminder.RegisterReminderFlags(v, flags), "Unexpected error")

	require.NoError(t, flags.Parse([]string{"--interval=1h", "--batch-size=200", "--min-elapsed=2h"}))

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(cfgbuf), "Unexpected error")

	cfg, err := config.ReadConfigFromViper[reminder.Config](v)
	require.NoError(t, err, "Unexpected error")

	require.Equal(t, parseTimeDuration(t, "1h"), cfg.RecurrenceConfig.Interval)
	require.Equal(t, 200, cfg.RecurrenceConfig.BatchSize)
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
	require.Equal(t, parseTimeDuration(t, "1h"), parseTimeDuration(t, v.GetString("recurrence.min_elapsed")))
	require.Equal(t, "watermill", v.GetString("events.sql.connection.dbname"))
	require.Equal(t, "localhost", v.GetString("events.sql.connection.dbhost"))
	require.Equal(t, "postgres", v.GetString("events.sql.connection.dbuser"))
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
	require.Equal(t, parseTimeDuration(t, "1h"), cfg.RecurrenceConfig.MinElapsed)
	require.Equal(t, "foobar", cfg.Database.Host)
}

func parseTimeDuration(t *testing.T, duration string) time.Duration {
	t.Helper()

	d, _ := time.ParseDuration(duration)
	return d
}
