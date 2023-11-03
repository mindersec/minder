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

package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"       // nolint
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/config"
	"github.com/stacklok/minder/internal/logger"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "migrate a down a database version",
	Long:  `Command to downgrade database`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.ReadConfigFromViper(viper.GetViper())
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
			fmt.Printf("Error getting flag yes: %v", err)
		}
		if !yes {
			fmt.Print("WARNING: Running this command will change the database structure. Are you want to continue? (y/n): ")
			var response string
			_, err := fmt.Scanln(&response)
			if err != nil {
				return fmt.Errorf("error reading response: %w", err)
			}

			if response == "n" {
				fmt.Println("Exiting...")
				return nil
			}
		}

		configPath := "file://" + filepath.Join(os.Getenv("KO_DATA_PATH"), "database/migrations")
		m, err := migrate.New(configPath, connString)
		if err != nil {
			fmt.Printf("Error while creating migration instance (%s): %v\n", configPath, err)
			os.Exit(1)
		}
		if err := m.Down(); err != nil {
			fmt.Printf("Error while migrating database: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Database migration down done with success. All Tables dropped")
		return nil
	},
}

func init() {
	migrateCmd.AddCommand(downCmd)
}
