// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reminder

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/pkg/config"
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
	// Entities holds per-entity-type overrides of the values above. Some entity
	// types need a slower cadence than repositories: for example, Docker Hub's
	// anonymous pull rate limit is roughly 17 pulls per hour, and every
	// reminder for an artifact triggers an evaluation that pulls from the
	// registry.
	Entities map[db.Entities]EntityRecurrenceConfig `mapstructure:"entities" viperdefault:"skip"`
}

// EntityRecurrenceConfig overrides RecurrenceConfig for one entity type.
// Unset fields fall back to the top-level RecurrenceConfig values.
type EntityRecurrenceConfig struct {
	// Enabled turns reminders for the entity type on or off. When unset,
	// only repositories are enabled.
	Enabled    *bool          `mapstructure:"enabled"`
	Interval   *time.Duration `mapstructure:"interval"`
	BatchSize  *int           `mapstructure:"batch_size"`
	MinElapsed *time.Duration `mapstructure:"min_elapsed"`
}

// EntityRecurrence is the effective recurrence configuration for one entity type.
type EntityRecurrence struct {
	Enabled    bool
	Interval   time.Duration
	BatchSize  int
	MinElapsed time.Duration
}

var validEntityTypes = []db.Entities{
	db.EntitiesRepository,
	db.EntitiesBuildEnvironment,
	db.EntitiesArtifact,
	db.EntitiesPullRequest,
	db.EntitiesRelease,
	db.EntitiesPipelineRun,
	db.EntitiesTaskRun,
	db.EntitiesBuild,
}

var entityRecurrenceFields = []string{"enabled", "interval", "batch_size", "min_elapsed"}

var defaultEnabledEntityTypes = []db.Entities{db.EntitiesRepository}

// ForEntityType returns the effective recurrence configuration for the given
// entity type, applying any per-type overrides on top of the top-level values.
func (r RecurrenceConfig) ForEntityType(entityType db.Entities) EntityRecurrence {
	out := EntityRecurrence{
		Enabled:    slices.Contains(defaultEnabledEntityTypes, entityType),
		Interval:   r.Interval,
		BatchSize:  r.BatchSize,
		MinElapsed: r.MinElapsed,
	}

	override, ok := r.Entities[entityType]
	if !ok {
		return out
	}
	if override.Enabled != nil {
		out.Enabled = *override.Enabled
	}
	if override.Interval != nil {
		out.Interval = *override.Interval
	}
	if override.BatchSize != nil {
		out.BatchSize = *override.BatchSize
	}
	if override.MinElapsed != nil {
		out.MinElapsed = *override.MinElapsed
	}
	return out
}

// Validate checks that the recurrence config is valid
func (r RecurrenceConfig) Validate() error {
	if r.MinElapsed < 0 {
		return fmt.Errorf("min_elapsed %s cannot be negative", r.MinElapsed)
	}

	if r.Interval < 0 {
		return fmt.Errorf("interval %s cannot be negative", r.Interval)
	}

	var errs []error
	for entityType, override := range r.Entities {
		if err := override.validate(entityType); err != nil {
			errs = append(errs, fmt.Errorf("entities.%s: %w", entityType, err))
		}
	}

	return errors.Join(errs...)
}

func (e EntityRecurrenceConfig) validate(entityType db.Entities) error {
	if !slices.Contains(validEntityTypes, entityType) {
		return fmt.Errorf("unknown entity type %q", entityType)
	}
	if e.Interval != nil && *e.Interval <= 0 {
		return fmt.Errorf("interval %s must be positive", *e.Interval)
	}
	if e.BatchSize != nil && *e.BatchSize <= 0 {
		return fmt.Errorf("batch_size %d must be positive", *e.BatchSize)
	}
	if e.MinElapsed != nil && *e.MinElapsed < 0 {
		return fmt.Errorf("min_elapsed %s cannot be negative", *e.MinElapsed)
	}
	return nil
}

// bindEntityRecurrenceEnv binds REMINDER_RECURRENCE_ENTITIES_<TYPE>_<FIELD>
// for every valid entity type. Unset variables resolve to nil, so they don't
// create an override entry.
func bindEntityRecurrenceEnv(v *viper.Viper) {
	for _, entityType := range validEntityTypes {
		for _, field := range entityRecurrenceFields {
			key := entityRecurrenceKey(entityType, field)
			if err := v.BindEnv(key); err != nil {
				panic(fmt.Sprintf("Failed to bind %q to env var: %v", key, err))
			}
		}
	}
}

func entityRecurrenceKey(entityType db.Entities, field string) string {
	return fmt.Sprintf("recurrence.entities.%s.%s", entityType, field)
}

// entityRecurrenceFlag sets per-entity-type overrides from a repeatable
// --entity-recurrence <type>.<field>=<value> flag. It writes to viper only
// when the flag is passed, because a flag bound with BindPFlag would supply
// its zero default for every type and field.
type entityRecurrenceFlag struct {
	v      *viper.Viper
	values []string
}

func (f *entityRecurrenceFlag) String() string {
	return strings.Join(f.values, ",")
}

func (*entityRecurrenceFlag) Type() string {
	return "type.field=value"
}

func (f *entityRecurrenceFlag) Set(value string) error {
	path, val, ok := strings.Cut(value, "=")
	if !ok {
		return fmt.Errorf("expected <type>.<field>=<value>, got %q", value)
	}
	entityType, field, ok := strings.Cut(path, ".")
	if !ok {
		return fmt.Errorf("expected <type>.<field>=<value>, got %q", value)
	}
	if !slices.Contains(validEntityTypes, db.Entities(entityType)) {
		return fmt.Errorf("unknown entity type %q", entityType)
	}
	if !slices.Contains(entityRecurrenceFields, field) {
		return fmt.Errorf("unknown field %q, expected one of %s", field, strings.Join(entityRecurrenceFields, ", "))
	}

	f.v.Set(entityRecurrenceKey(db.Entities(entityType), field), val)
	f.values = append(f.values, value)
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
	err = config.BindConfigFlag(
		v, flags, viperPath, "min-elapsed", v.GetString(viperPath),
		"Minimum time after last update before sending a reminder", flags.String,
	)
	if err != nil {
		return err
	}

	flags.Var(&entityRecurrenceFlag{v: v}, "entity-recurrence",
		"Per-entity-type recurrence override, e.g. artifact.batch_size=10 (repeatable)")
	return nil
}
