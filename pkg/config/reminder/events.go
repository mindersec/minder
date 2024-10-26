// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reminder

import (
	"github.com/mindersec/minder/pkg/config"
)

// EventConfig is the configuration for reminder's eventing system.
type EventConfig struct {
	Driver       string            `mapstructure:"driver" default:"sql"`
	SQLPubConfig SQLPubConfig      `mapstructure:"sql"`
	NatsConfig   config.NatsConfig `mapstructure:"nats"`
}

// SQLPubConfig is the configuration for the SQL publisher
type SQLPubConfig struct {
	// Connection is the configuration for the SQL event driver
	//
	// nolint: lll
	Connection config.DatabaseConfig `mapstructure:"connection" default:"{\"dbname\":\"reminder\",\"dbhost\":\"reminder-event-postgres\"}"`
}
