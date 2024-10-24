// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package app provides the entrypoint for the minder migrations
package app

import (
	"context"
	"fmt"
	"os"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"       // nolint
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/database"
	"github.com/mindersec/minder/pkg/config"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "get the db version",
	Long:  `Command to get the database version`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.ReadConfigFromViper[serverconfig.Config](viper.GetViper())
		if err != nil {
			return fmt.Errorf("unable to read config: %w", err)
		}

		ctx := serverconfig.LoggerFromConfigFlags(cfg.LoggingConfig).WithContext(context.Background())

		// Database configuration
		dbConn, connString, err := cfg.Database.GetDBConnection(ctx)
		if err != nil {
			return fmt.Errorf("unable to connect to database: %w", err)
		}
		defer dbConn.Close()

		m, err := database.NewFromConnectionString(connString)
		if err != nil {
			cmd.Printf("Error while creating migration instance: %v\n", err)
			os.Exit(1)
		}

		version, dirty, err := m.Version()
		if err != nil {
			cmd.Printf("Error while getting migration version: %v\n", err)
			os.Exit(1)
		}

		cmd.Printf("Version=%v dirty=%v\n", version, dirty)
		return nil
	},
}

func init() {
	migrateCmd.AddCommand(versionCmd)
}
