// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reminder

import (
	"github.com/mindersec/minder/internal/config"
)

// EventConfig is the configuration for reminder's eventing system.
type EventConfig struct {
	// Connection is the configuration for the SQL event driver
	//
	// nolint: lll
	Connection config.DatabaseConfig `mapstructure:"sql_connection" default:"{\"dbname\":\"reminder\",\"dbhost\":\"reminder-event-postgres\"}"`
}
