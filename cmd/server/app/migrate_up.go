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
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"       // nolint
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/authz"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/logger"
)

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up",
	Short: "migrate the database to the latest version",
	Long:  `Command to upgrade database`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := serverconfig.ReadConfigFromViper(viper.GetViper())
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

		yes, err := cmd.Flags().GetBool("yes")
		if err != nil {
			cmd.Printf("Error while getting yes flag: %v", err)
		}
		if !yes {
			cmd.Print("WARNING: Running this command will change the database structure. Are you want to continue? (y/n): ")
			var response string
			_, err := fmt.Scanln(&response)
			if err != nil {
				return fmt.Errorf("error while reading user input: %w", err)
			}

			if response == "n" {
				cmd.Printf("Exiting...")
				return nil
			}
		}

		configPath := getMigrateConfigPath()
		m, err := migrate.New(configPath, connString)
		if err != nil {
			cmd.Printf("Error while creating migration instance (%s): %v\n", configPath, err)
			os.Exit(1)
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
				cmd.Printf("Error while migrating database: %v\n", err)
				os.Exit(1)
			} else {
				cmd.Println("Database already up-to-date")
			}
		}

		cmd.Println("Database migration completed successfully")

		cmd.Println("Ensuring authorization store...")

		authzw, err := authz.NewAuthzClient(&cfg.Authz)
		if err != nil {
			return fmt.Errorf("error while creating authz client: %w", err)
		}

		if !authzw.StoreIDProvided() {
			if err := ensureAuthzStore(ctx, cmd, authzw); err != nil {
				return err
			}
		}

		return nil
	},
}

func ensureAuthzStore(ctx context.Context, cmd *cobra.Command, authzw *authz.ClientWrapper) error {
	storeName := authzw.GetConfig().StoreName
	storeID, err := authzw.FindStoreByName(ctx)
	if err != nil && !errors.Is(err, authz.ErrStoreNotFound) {
		return err
	} else if errors.Is(err, authz.ErrStoreNotFound) {
		cmd.Printf("Creating authz store %s\n", storeName)
		id, err := authzw.CreateStore(ctx)
		if err != nil {
			return err
		}
		cmd.Printf("Created authz store %s/%s\n", id, storeName)
		return nil
	}

	cmd.Printf("Not creating store. Found store with name '%s' and ID '%s'.",
		storeName, storeID)

	return nil
}

func init() {
	migrateCmd.AddCommand(upCmd)
}
