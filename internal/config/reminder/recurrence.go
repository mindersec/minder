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

package reminder

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/config"
)

// RecurrenceConfig contains the configuration for the reminder recurrence
type RecurrenceConfig struct {
	// Interval is the time between reminders
	Interval time.Duration `mapstructure:"interval" default:"1h"`
	// BatchSize is the number of reminders to process at once. Batch size cannot be less than
	// MaxPerProject * MinProjectFetchLimit.
	BatchSize int `mapstructure:"batch_size" default:"100"`
	// MinElapsed is the minimum time after last update before sending a reminder
	MinElapsed time.Duration `mapstructure:"min_elapsed" default:"1h"`
}

// Validate checks that the recurrence config is valid
func (r RecurrenceConfig) Validate() error {
	if r.MinElapsed < 0 {
		return fmt.Errorf("min_elapsed %s cannot be negative", r.MinElapsed)
	}

	if r.Interval < 0 {
		return fmt.Errorf("interval %s cannot be negative", r.Interval)
	}

	return nil
}

func registerRecurrenceFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	viperPath := "recurrence.interval"
	err := config.BindConfigFlagWithShort(
		v, flags, viperPath, "interval", "i", v.GetString(viperPath),
		"Interval between reminders", flags.StringP)
	if err != nil {
		return err
	}

	viperPath = "recurrence.batch_size"
	err = config.BindConfigFlagWithShort(
		v, flags, "recurrence.batch_size", "batch-size", "b", v.GetInt(viperPath),
		"Number of reminders to process at once", flags.IntP)
	if err != nil {
		return err
	}

	viperPath = "recurrence.min_elapsed"
	return config.BindConfigFlag(
		v, flags, viperPath, "min-elapsed", v.GetString(viperPath),
		"Minimum time after last update before sending a reminder", flags.String,
	)
}
