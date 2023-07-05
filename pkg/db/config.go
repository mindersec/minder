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

package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/pkg/util"
)

// Config is the configuration for the database
type Config struct {
	Host          string `mapstructure:"dbhost"`
	Port          int    `mapstructure:"dbport"`
	User          string `mapstructure:"dbuser"`
	Password      string `mapstructure:"dbpass"`
	Name          string `mapstructure:"dbname"`
	SSLMode       string `mapstructure:"sslmode"`
	EncryptionKey string `mapstructure:"encryption_key"`
}

// GetDBURI returns the database URI
func (c *Config) GetDBURI() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode)
}

// GetDBConnection returns a connection to the database
func (c *Config) GetDBConnection() (*sql.DB, string, error) {
	conn, err := sql.Open("postgres", c.GetDBURI())
	if err != nil {
		return nil, "", err
	}

	// Ensure we actually connected to the database, per Go docs
	if err := conn.Ping(); err != nil {
		//nolint:gosec // Not much we can do about an error here.
		conn.Close()
		return nil, "", err
	}

	log.Println("Connected to DB")
	return conn, c.GetDBURI(), err
}

// RegisterFlags registers the flags for the database configuration
func RegisterFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	err := util.BindConfigFlagWithShort(
		v, flags, "database.dbhost", "db-host", "H", "localhost", "Database host", flags.StringP)
	if err != nil {
		return err
	}

	err = util.BindConfigFlag(
		v, flags, "database.dbport", "db-port", 5432, "Database port", flags.Int)
	if err != nil {
		return err
	}

	err = util.BindConfigFlagWithShort(
		v, flags, "database.dbuser", "db-user", "u", "postgres", "Database user", flags.StringP)
	if err != nil {
		return err
	}

	err = util.BindConfigFlagWithShort(
		v, flags, "database.dbpass", "db-pass", "P", "postgres", "Database password", flags.StringP)
	if err != nil {
		return err
	}

	err = util.BindConfigFlagWithShort(
		v, flags, "database.dbname", "db-name", "d", "mediator", "Database name", flags.StringP)
	if err != nil {
		return err
	}

	return util.BindConfigFlagWithShort(
		v, flags, "database.sslmode", "db-sslmode", "s", "disable", "Database sslmode", flags.StringP)
}
