// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reminder_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/db"
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

func TestRecurrenceForEntityType(t *testing.T) {
	t.Parallel()

	base := reminder.RecurrenceConfig{
		Interval:   time.Hour,
		BatchSize:  100,
		MinElapsed: time.Hour,
	}
	topLevel := func(enabled bool) reminder.EntityRecurrence {
		return reminder.EntityRecurrence{
			Enabled: enabled, Interval: time.Hour, BatchSize: 100, MinElapsed: time.Hour,
		}
	}

	tests := []struct {
		name       string
		entities   map[db.Entities]reminder.EntityRecurrenceConfig
		entityType db.Entities
		want       reminder.EntityRecurrence
	}{
		{
			name:       "no entities section keeps repositories enabled with top-level values",
			entityType: db.EntitiesRepository,
			want:       topLevel(true),
		},
		{
			name:       "no entities section leaves other types disabled",
			entityType: db.EntitiesArtifact,
			want:       topLevel(false),
		},
		{
			name:       "empty override falls back to top-level values",
			entities:   map[db.Entities]reminder.EntityRecurrenceConfig{db.EntitiesRepository: {}},
			entityType: db.EntitiesRepository,
			want:       topLevel(true),
		},
		{
			name: "partial override only replaces the fields it sets",
			entities: map[db.Entities]reminder.EntityRecurrenceConfig{
				db.EntitiesArtifact: {BatchSize: ptr(10)},
			},
			entityType: db.EntitiesArtifact,
			want: reminder.EntityRecurrence{
				Enabled: false, Interval: time.Hour, BatchSize: 10, MinElapsed: time.Hour,
			},
		},
		{
			name: "explicit override wins",
			entities: map[db.Entities]reminder.EntityRecurrenceConfig{
				db.EntitiesArtifact: {
					Enabled:    ptr(true),
					Interval:   ptr(6 * time.Hour),
					BatchSize:  ptr(10),
					MinElapsed: ptr(12 * time.Hour),
				},
			},
			entityType: db.EntitiesArtifact,
			want: reminder.EntityRecurrence{
				Enabled: true, Interval: 6 * time.Hour, BatchSize: 10, MinElapsed: 12 * time.Hour,
			},
		},
		{
			name: "repositories can be disabled explicitly",
			entities: map[db.Entities]reminder.EntityRecurrenceConfig{
				db.EntitiesRepository: {Enabled: ptr(false)},
			},
			entityType: db.EntitiesRepository,
			want:       topLevel(false),
		},
		{
			name: "override for another type does not leak",
			entities: map[db.Entities]reminder.EntityRecurrenceConfig{
				db.EntitiesArtifact: {Enabled: ptr(true), BatchSize: ptr(10)},
			},
			entityType: db.EntitiesRepository,
			want:       topLevel(true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := base
			cfg.Entities = tt.entities
			require.Equal(t, tt.want, cfg.ForEntityType(tt.entityType))
		})
	}
}

func TestValidateEntityRecurrence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entities map[db.Entities]reminder.EntityRecurrenceConfig
		errMsg   string
	}{
		{
			name:     "empty override is valid",
			entities: map[db.Entities]reminder.EntityRecurrenceConfig{db.EntitiesArtifact: {}},
		},
		{
			name: "full override is valid",
			entities: map[db.Entities]reminder.EntityRecurrenceConfig{
				db.EntitiesArtifact: {
					Enabled: ptr(true), Interval: ptr(time.Hour), BatchSize: ptr(1), MinElapsed: ptr(time.Duration(0)),
				},
			},
		},
		{
			name:     "unknown entity type",
			entities: map[db.Entities]reminder.EntityRecurrenceConfig{"container": {}},
			errMsg:   `entities.container: unknown entity type "container"`,
		},
		{
			name:     "zero interval",
			entities: map[db.Entities]reminder.EntityRecurrenceConfig{db.EntitiesArtifact: {Interval: ptr(time.Duration(0))}},
			errMsg:   "entities.artifact: interval 0s must be positive",
		},
		{
			name:     "negative interval",
			entities: map[db.Entities]reminder.EntityRecurrenceConfig{db.EntitiesArtifact: {Interval: ptr(-time.Hour)}},
			errMsg:   "entities.artifact: interval -1h0m0s must be positive",
		},
		{
			name:     "zero batch size",
			entities: map[db.Entities]reminder.EntityRecurrenceConfig{db.EntitiesArtifact: {BatchSize: ptr(0)}},
			errMsg:   "entities.artifact: batch_size 0 must be positive",
		},
		{
			name:     "negative min elapsed",
			entities: map[db.Entities]reminder.EntityRecurrenceConfig{db.EntitiesArtifact: {MinElapsed: ptr(-time.Hour)}},
			errMsg:   "entities.artifact: min_elapsed -1h0m0s cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := reminder.Config{
				RecurrenceConfig: reminder.RecurrenceConfig{
					Interval: time.Hour, BatchSize: 100, MinElapsed: time.Hour, Entities: tt.entities,
				},
				EventConfig: serverconfig.EventConfig{Driver: constants.SQLDriver},
			}
			err := cfg.Validate()
			if tt.errMsg != "" {
				require.ErrorContains(t, err, tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestReadConfigWithoutEntitiesIsBackwardCompatible(t *testing.T) {
	t.Parallel()

	v := viper.New()
	reminder.SetViperDefaults(v)
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	require.NoError(t, reminder.RegisterReminderFlags(v, flags))
	require.NoError(t, flags.Parse(nil))

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(bytes.NewBufferString(`---
recurrence:
  interval: "1m"
  batch_size: 50
  min_elapsed: "2h"
`)))

	cfg, err := config.ReadConfigFromViper[reminder.Config](v)
	require.NoError(t, err)
	require.NoError(t, cfg.RecurrenceConfig.Validate())
	require.Empty(t, cfg.RecurrenceConfig.Entities)
	require.Equal(t, reminder.EntityRecurrence{
		Enabled: true, Interval: time.Minute, BatchSize: 50, MinElapsed: 2 * time.Hour,
	}, cfg.RecurrenceConfig.ForEntityType(db.EntitiesRepository))
}

func TestReadConfigWithEntities(t *testing.T) {
	t.Parallel()

	v := viper.New()
	reminder.SetViperDefaults(v)
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(bytes.NewBufferString(`---
recurrence:
  interval: "1h"
  batch_size: 100
  min_elapsed: "1h"
  entities:
    artifact:
      enabled: true
      interval: "6h"
      batch_size: 5
    repository:
      min_elapsed: "30m"
`)))

	cfg, err := config.ReadConfigFromViper[reminder.Config](v)
	require.NoError(t, err)
	require.NoError(t, cfg.RecurrenceConfig.Validate())

	require.Equal(t, reminder.EntityRecurrence{
		Enabled: true, Interval: 6 * time.Hour, BatchSize: 5, MinElapsed: time.Hour,
	}, cfg.RecurrenceConfig.ForEntityType(db.EntitiesArtifact))
	require.Equal(t, reminder.EntityRecurrence{
		Enabled: true, Interval: time.Hour, BatchSize: 100, MinElapsed: 30 * time.Minute,
	}, cfg.RecurrenceConfig.ForEntityType(db.EntitiesRepository))
}

func TestEntityRecurrenceFlag(t *testing.T) {
	t.Parallel()

	v := viper.New()
	reminder.SetViperDefaults(v)
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	require.NoError(t, reminder.RegisterReminderFlags(v, flags))
	require.NoError(t, flags.Parse([]string{
		"--entity-recurrence=artifact.enabled=true",
		"--entity-recurrence", "artifact.batch_size=7",
	}))

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(bytes.NewBufferString(`---
recurrence:
  entities:
    artifact:
      batch_size: 50
      interval: "3h"
`)))

	cfg, err := config.ReadConfigFromViper[reminder.Config](v)
	require.NoError(t, err)
	require.Len(t, cfg.RecurrenceConfig.Entities, 1)
	require.Equal(t, reminder.EntityRecurrence{
		Enabled: true, Interval: 3 * time.Hour, BatchSize: 7, MinElapsed: time.Hour,
	}, cfg.RecurrenceConfig.ForEntityType(db.EntitiesArtifact))
}

func TestEntityRecurrenceFlagRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	for _, arg := range []string{
		"artifact.batch_size",
		"artifact=5",
		"container.batch_size=5",
		"artifact.cadence=5",
	} {
		t.Run(arg, func(t *testing.T) {
			t.Parallel()

			v := viper.New()
			flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
			flags.SetOutput(io.Discard)
			require.NoError(t, reminder.RegisterReminderFlags(v, flags))
			require.Error(t, flags.Parse([]string{"--entity-recurrence=" + arg}))
		})
	}
}

// TestOverrideEntityRecurrenceByEnvVar is not parallel because it sets environment variables.
//
// nolint: paralleltest
func TestOverrideEntityRecurrenceByEnvVar(t *testing.T) {
	t.Setenv("REMINDER_RECURRENCE_ENTITIES_ARTIFACT_ENABLED", "true")
	t.Setenv("REMINDER_RECURRENCE_ENTITIES_ARTIFACT_INTERVAL", "6h")

	v := viper.New()
	reminder.SetViperDefaults(v)
	v.AutomaticEnv()
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(bytes.NewBufferString(`---
recurrence:
  batch_size: 100
  entities:
    artifact:
      batch_size: 5
`)))

	cfg, err := config.ReadConfigFromViper[reminder.Config](v)
	require.NoError(t, err)
	require.Len(t, cfg.RecurrenceConfig.Entities, 1)
	require.Equal(t, reminder.EntityRecurrence{
		Enabled: true, Interval: 6 * time.Hour, BatchSize: 5, MinElapsed: time.Hour,
	}, cfg.RecurrenceConfig.ForEntityType(db.EntitiesArtifact))
}

func ptr[T any](v T) *T {
	return &v
}

func parseTimeDuration(t *testing.T, duration string) time.Duration {
	t.Helper()

	d, _ := time.ParseDuration(duration)
	return d
}
