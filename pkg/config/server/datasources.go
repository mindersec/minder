// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

import "time"

// DataSourceConfig contains server-controlled data source limits.
type DataSourceConfig struct {
	REST RESTDataSourceConfig `mapstructure:"rest"`
}

// RESTDataSourceConfig contains resource limits for REST data source requests.
type RESTDataSourceConfig struct {
	RequestTimeout   time.Duration `mapstructure:"request_timeout" default:"5s"`
	MaxResponseBytes int64         `mapstructure:"max_response_bytes" default:"1048576"`
}
