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
	"testing"

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
		name   string
		config *reminder.Config
		errMsg string
	}{
		{
			name: "ValidValues",
			config: &reminder.Config{
				RecurrenceConfig: reminder.RecurrenceConfig{
					Interval:             "1h",
					BatchSize:            100,
					MaxPerProject:        10,
					MinProjectFetchLimit: 5,
					MinElapsed:           "1h",
				},
				EventConfig: reminder.EventConfig{
					Connection: config.DatabaseConfig{
						Port: 8080,
					},
				},
			},
		},
		{
			name: "Empty Event Config",
			config: &reminder.Config{
				RecurrenceConfig: reminder.RecurrenceConfig{
					Interval:             "1h",
					BatchSize:            100,
					MaxPerProject:        10,
					MinProjectFetchLimit: 5,
					MinElapsed:           "1h",
				},
				EventConfig: reminder.EventConfig{
					Connection: config.DatabaseConfig{},
				},
			},
			errMsg: "event config is empty, required for sending events to minder server",
		},
		{
			name: "InvalidInterval",
			config: &reminder.Config{
				RecurrenceConfig: reminder.RecurrenceConfig{
					Interval:             "1x",
					BatchSize:            100,
					MaxPerProject:        10,
					MinProjectFetchLimit: 5,
					MinElapsed:           "1h",
				},
			},
			errMsg: "invalid interval: 1x",
		},
		{
			name: "InvalidMinElapsed",
			config: &reminder.Config{
				RecurrenceConfig: reminder.RecurrenceConfig{
					Interval:             "1h",
					BatchSize:            100,
					MaxPerProject:        10,
					MinProjectFetchLimit: 5,
					MinElapsed:           "1x",
				},
			},
			errMsg: "invalid min_elapsed: 1x",
		},
		{
			name: "InvalidBatchSize",
			config: &reminder.Config{
				RecurrenceConfig: reminder.RecurrenceConfig{
					Interval:             "1h",
					BatchSize:            10,
					MaxPerProject:        10,
					MinProjectFetchLimit: 5,
					MinElapsed:           "1h",
				},
			},
			errMsg: "batch_size 10 cannot be less than max_per_project(10)*min_project_fetch_limit(5)=50",
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := reminder.ValidateConfig(tt.config)
			if tt.errMsg != "" {
				assert.EqualError(t, err, tt.errMsg)
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
  max_per_project: 10
  min_project_fetch_limit: 10
  min_elapsed: "1h"
`

	cfgbuf := bytes.NewBufferString(cfgstr)

	v := viper.New()

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(cfgbuf), "Unexpected error")

	cfg, err := config.ReadConfigFromViper[reminder.Config](v)
	require.NoError(t, err, "Unexpected error")

	require.Equal(t, "1m", cfg.RecurrenceConfig.Interval)
	require.Equal(t, 100, cfg.RecurrenceConfig.BatchSize)
	require.Equal(t, 10, cfg.RecurrenceConfig.MaxPerProject)
	require.Equal(t, 10, cfg.RecurrenceConfig.MinProjectFetchLimit)
	require.Equal(t, "1h", cfg.RecurrenceConfig.MinElapsed)
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
`

	cfgbuf := bytes.NewBufferString(cfgstr)

	v := viper.New()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)

	require.NoError(t, reminder.RegisterReminderFlags(v, flags), "Unexpected error")

	require.NoError(t, flags.Parse([]string{"--interval=1h", "--batch-size=200", "--max-per-project=20", "--min-project-fetch-limit=20", "--min-elapsed=2h"}))

	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(cfgbuf), "Unexpected error")

	cfg, err := config.ReadConfigFromViper[reminder.Config](v)
	require.NoError(t, err, "Unexpected error")

	require.Equal(t, "1h", cfg.RecurrenceConfig.Interval)
	require.Equal(t, 200, cfg.RecurrenceConfig.BatchSize)
	require.Equal(t, 20, cfg.RecurrenceConfig.MaxPerProject)
	require.Equal(t, 20, cfg.RecurrenceConfig.MinProjectFetchLimit)
	require.Equal(t, "2h", cfg.RecurrenceConfig.MinElapsed)
}
