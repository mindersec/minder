// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package reminder contains configuration options for the reminder service.
package reminder

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/pkg/config"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer/constants"
)

// Config contains the configuration for the reminder service
type Config struct {
	Database         config.DatabaseConfig           `mapstructure:"database"`
	RecurrenceConfig RecurrenceConfig                `mapstructure:"recurrence"`
	EventConfig      serverconfig.EventConfig        `mapstructure:"events"`
	LoggingConfig    LoggingConfig                   `mapstructure:"logging"`
	MetricsConfig    serverconfig.MetricsConfig      `mapstructure:"metrics"`
	MetricServer     serverconfig.MetricServerConfig `mapstructure:"metric_server" default:"{\"port\":\"9091\"}"`
}

// Validate validates the configuration
func (c Config) Validate() error {
	err := c.RecurrenceConfig.Validate()
	if err != nil {
		return err
	}

	err = validateEventConfig(c.EventConfig)
	if err != nil {
		return err
	}

	return nil
}

// SetViperDefaults sets the default values for the configuration to be picked up by viper
func SetViperDefaults(v *viper.Viper) {
	v.SetEnvPrefix("reminder")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	config.SetViperStructDefaults(v, "", Config{})
}

// RegisterReminderFlags registers the flags for the minder cli
func RegisterReminderFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	viperPath := "logging.level"
	if err := config.BindConfigFlag(v, flags, viperPath, "logging-level",
		v.GetString(viperPath), "Logging level for reminder", flags.String); err != nil {
		return err
	}

	if err := config.RegisterDatabaseFlags(v, flags); err != nil {
		return err
	}

	return registerRecurrenceFlags(v, flags)
}

func validateEventConfig(cfg serverconfig.EventConfig) error {
	switch cfg.Driver {
	case constants.NATSDriver:
	case constants.SQLDriver:
	default:
		return fmt.Errorf("events.driver %s is not supported", cfg.Driver)
	}

	return nil
}
