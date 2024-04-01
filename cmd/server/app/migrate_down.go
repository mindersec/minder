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

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"       // nolint
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/database"
	"github.com/stacklok/minder/internal/config"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/logger"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "migrate a down a database version",
	Long:  `Command to downgrade database`,
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

		yes, err := cmd.Flags().GetBool("yes")
		if err != nil {
			cmd.Printf("Error getting flag yes: %v", err)
		}
		if !yes {
			cmd.Print("WARNING: Running this command will change the database structure. Are you want to continue? (y/n): ")
			var response string
			_, err := fmt.Scanln(&response)
			if err != nil {
				return fmt.Errorf("error reading response: %w", err)
			}

			if response == "n" {
				cmd.Println("Exiting...")
				return nil
			}
		}

		m, err := database.NewFromConnectionString(connString)
		if err != nil {
			cmd.Printf("Error while creating migration instance: %v\n", err)
			os.Exit(1)
		}

		var usteps uint
		usteps, err = cmd.Flags().GetUint("num-steps")
		if err != nil {
			cmd.Printf("Error while getting num-steps flag: %v", err)
		}

		if usteps == 0 {
			err = m.Down()
		} else {
			err = m.Steps(-1 * int(usteps))
		}

		if err != nil {
			cmd.Printf("Error while migrating database: %v\n", err)
			os.Exit(1)
		}

		cmd.Println("Database migration down done with success")
		return nil
	},
}

func init() {
	migrateCmd.AddCommand(downCmd)
}
