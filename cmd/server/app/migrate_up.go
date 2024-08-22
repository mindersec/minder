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

// Package app provides the entrypoint for the minder migrations
package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"       // nolint
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/database"
	"github.com/stacklok/minder/internal/authz"
	"github.com/stacklok/minder/internal/config"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/logger"
)

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up",
	Short: "migrate the database to the latest version",
	Long:  `Command to upgrade database`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.ReadConfigFromViper[serverconfig.Config](viper.GetViper())
		if err != nil {
			return fmt.Errorf("unable to read config: %w", err)
		}

		ctx := logger.FromFlags(cfg.LoggingConfig).WithContext(context.Background())

		// Database configuration
		dbConn, connString, err := cfg.Database.GetDBConnection(ctx)
		if err != nil {
			return fmt.Errorf("unable to connect to database: %w", err)
		}
		defer dbConn.Close()

		yes := confirm(cmd, "Running this command will change the database structure")
		if !yes {
			return nil
		}

		m, err := database.NewFromConnectionString(connString)
		if err != nil {
			cliErrorf(cmd, "Error while creating migration instance: %v\n", err)
		}

		var usteps uint
		usteps, err = cmd.Flags().GetUint("num-steps")
		if err != nil {
			cmd.Printf("Error while getting num-steps flag: %v", err)
		}

		if usteps == 0 {
			err = m.Up()
		} else {
			err = m.Steps(int(usteps))
		}

		if err != nil {
			if !errors.Is(err, migrate.ErrNoChange) {
				cliErrorf(cmd, "Error while migrating database: %v\n", err)
			} else {
				cmd.Println("Database already up-to-date")
			}
		}

		cmd.Println("Database migration completed successfully")

		version, dirty, err := m.Version()
		if err != nil {
			cmd.Printf("Error while getting migration version: %v\n", err)
			// not fatal
		} else {
			cmd.Printf("Version=%v dirty=%v\n", version, dirty)
		}

		cmd.Println("Ensuring authorization store...")
		l := zerolog.Ctx(ctx)

		authzw, err := authz.NewAuthzClient(&cfg.Authz, l)
		if err != nil {
			return fmt.Errorf("error while creating authz client: %w", err)
		}

		if err := authzw.MigrateUp(ctx); err != nil {
			return fmt.Errorf("error while running authz migrations: %w", err)
		}

		if err := authzw.PrepareForRun(ctx); err != nil {
			return fmt.Errorf("error preparing authz client: %w", err)
		}

		cmd.Println("Performing entity migrations...")
		store := db.NewStore(dbConn)

		if err := store.TemporaryPopulateRepositories(ctx); err != nil {
			cmd.Printf("Error while populating entities table with repos: %v\n", err)
		}

		if err := store.TemporaryPopulateArtifacts(ctx); err != nil {
			cmd.Printf("Error while populating entities table with artifacts: %v\n", err)
		}

		if err := store.TemporaryPopulatePullRequests(ctx); err != nil {
			cmd.Printf("Error while populating entities table with pull requests: %v\n", err)
		}

		if err := store.TemporaryPopulateEvaluationHistory(ctx); err != nil {
			cmd.Printf("Error while populating entities table with evaluation history: %v\n", err)
		}

		if err := store.TemporaryPopulateRuleTypeState(ctx); err != nil {
			cmd.Printf("Error updating status of existing rule types: %v\n", err)
		}

		return nil
	},
}

func init() {
	migrateCmd.AddCommand(upCmd)
}
