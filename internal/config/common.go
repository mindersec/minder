//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/rs/zerolog"
	"github.com/signalfx/splunk-otel-go/instrumentation/database/sql/splunksql"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/constants"
)

// DatabaseConfig is the configuration for the database
type DatabaseConfig struct {
	Host     string `mapstructure:"dbhost" default:"localhost"`
	Port     int    `mapstructure:"dbport" default:"5432"`
	User     string `mapstructure:"dbuser" default:"postgres"`
	Password string `mapstructure:"dbpass" default:"postgres"`
	Name     string `mapstructure:"dbname" default:"minder"`
	SSLMode  string `mapstructure:"sslmode" default:"disable"`
}

// GetDBConnection returns a connection to the database
func (c *DatabaseConfig) GetDBConnection(ctx context.Context) (*sql.DB, string, error) {
	uri := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, url.QueryEscape(c.Password), c.Host, c.Port, c.Name, c.SSLMode)

	conn, err := splunksql.Open("postgres", uri)
	if err != nil {
		return nil, "", err
	}

	for i := 0; i < 8; i++ {
		// Ensure we actually connected to the database, per Go docs
		err = conn.Ping()
		if err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msgf("Unable to initialize connection to DB, retry %d", i)
			time.Sleep(1 * time.Second)
			continue
		}
		zerolog.Ctx(ctx).Info().Msg("Connected to DB")
		return conn, uri, err
	}

	//nolint:gosec // Not much we can do about an error here.
	conn.Close()
	return nil, "", err
}

// RegisterDatabaseFlags registers the flags for the database configuration
func RegisterDatabaseFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	err := BindConfigFlagWithShort(
		v, flags, "database.dbhost", "db-host", "H", "localhost", "Database host", flags.StringP)
	if err != nil {
		return err
	}

	err = BindConfigFlag(
		v, flags, "database.dbport", "db-port", 5432, "Database port", flags.Int)
	if err != nil {
		return err
	}

	err = BindConfigFlagWithShort(
		v, flags, "database.dbuser", "db-user", "u", "postgres", "Database user", flags.StringP)
	if err != nil {
		return err
	}

	err = BindConfigFlagWithShort(
		v, flags, "database.dbpass", "db-pass", "P", "postgres", "Database password", flags.StringP)
	if err != nil {
		return err
	}

	err = BindConfigFlagWithShort(
		v, flags, "database.dbname", "db-name", "d", "minder", "Database name", flags.StringP)
	if err != nil {
		return err
	}

	return BindConfigFlagWithShort(
		v, flags, "database.sslmode", "db-sslmode", "s", "disable", "Database sslmode", flags.StringP)
}

// GRPCClientConfig is the configuration for a service to connect to minder gRPC server
type GRPCClientConfig struct {
	// Host is the host to connect to
	Host string `mapstructure:"host" default:"api.stacklok.com"`

	// Port is the port to connect to
	Port int `mapstructure:"port" default:"443"`

	// Insecure is whether to allow establishing insecure connections
	Insecure bool `mapstructure:"insecure" default:"false"`
}

// RegisterGRPCClientConfigFlags registers the flags for the gRPC client
func RegisterGRPCClientConfigFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	err := BindConfigFlag(v, flags, "grpc_server.host", "grpc-host", constants.MinderGRPCHost,
		"Server host", flags.String)
	if err != nil {
		return err
	}

	err = BindConfigFlag(v, flags, "grpc_server.port", "grpc-port", 443,
		"Server port", flags.Int)
	if err != nil {
		return err
	}

	return BindConfigFlag(v, flags, "grpc_server.insecure", "grpc-insecure", false,
		"Allow establishing insecure connections", flags.Bool)
}
