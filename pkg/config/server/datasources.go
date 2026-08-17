// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"math"
	"time"

	"github.com/dustin/go-humanize"
)

// DataSourceConfig contains server-controlled data source limits.
type DataSourceConfig struct {
	REST RESTDataSourceConfig `mapstructure:"rest"`
}

// RESTDataSourceConfig contains resource limits for REST data source requests.
type RESTDataSourceConfig struct {
	RequestTimeout   time.Duration `mapstructure:"request_timeout" default:"0s"`
	MaxResponseBytes string        `mapstructure:"max_response_bytes"`
}

// GetMaxResponseBytes parses the configured human-readable response size.
// Zero means the REST data source should use its code-defined default.
func (c *RESTDataSourceConfig) GetMaxResponseBytes() (int64, error) {
	if c.MaxResponseBytes == "" {
		return 0, nil
	}

	size, err := humanize.ParseBytes(c.MaxResponseBytes)
	if err != nil {
		return 0, fmt.Errorf("parse max_response_bytes %q: %w", c.MaxResponseBytes, err)
	}
	if size > uint64(math.MaxInt64) {
		return 0, fmt.Errorf("max_response_bytes %q exceeds the supported limit", c.MaxResponseBytes)
	}

	return int64(size), nil
}
