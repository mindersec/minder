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
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "migrate a down a database version",
	Long:  `Command to downgrade database`,
	Run: func(cmd *cobra.Command, args []string) {

		dbhost := viper.GetString("database.dbhost")
		dbport := viper.GetString("database.dbport")
		dbuser := viper.GetString("database.dbuser")
		dbpass := viper.GetString("database.dbpass")
		dbname := viper.GetString("database.dbname")
		sslmode := viper.GetString("database.sslmode")

		if cmd.Flags().Changed("dbhost") {
			dbhost, _ = cmd.Flags().GetString("dbhost")
		}
		if cmd.Flags().Changed("dbport") {
			dbport, _ = cmd.Flags().GetString("dbport")
		}
		if cmd.Flags().Changed("dbuser") {
			dbuser, _ = cmd.Flags().GetString("dbuser")
		}
		if cmd.Flags().Changed("dbpass") {
			dbpass, _ = cmd.Flags().GetString("dbpass")
		}
		if cmd.Flags().Changed("dbname") {
			dbname, _ = cmd.Flags().GetString("dbname")
		}
		if cmd.Flags().Changed("sslmode") {
			sslmode, _ = cmd.Flags().GetString("sslmode")
		}

		connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbuser, dbpass, dbhost, dbport, dbname, sslmode)

		yes, err := cmd.Flags().GetBool("yes")
		if err != nil {
			fmt.Printf("Error getting flag yes: %v", err)
		}
		if !yes {
			fmt.Print("WARNING: Running this command will change the database structure. Are you want to continue? (y/n): ")
			var response string
			fmt.Scanln(&response)

			if response == "n" {
				fmt.Println("Exiting...")
				os.Exit(0)
			}
		}

		m, err := migrate.New(
			"file://database/migrations",
			connString)
		if err != nil {
			fmt.Printf("Error while creating migration instance: %v", err)
		}
		if err := m.Down(); err != nil {
			fmt.Printf("Error while migrating database: %v", err)
		}

		fmt.Println("Database migration down done with success. All Tables dropped")
	},
}

func init() {
	migrateCmd.AddCommand(downCmd)
}
