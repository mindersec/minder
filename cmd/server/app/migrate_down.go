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
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"       // nolint
	"github.com/spf13/cobra"
	"github.com/stacklok/mediator/pkg/util"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "migrate a down a database version",
	Long:  `Command to downgrade database`,
	Run: func(cmd *cobra.Command, args []string) {

		dbConn, connString, err := util.GetDbConnectionFromConfig(cmd)
		if err != nil {
			fmt.Printf("Unable to connect to database: %v\n", err)
			os.Exit(1)
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
				fmt.Printf("Error reading response: %v\n", err)
				os.Exit(1)
			}

			if response == "n" {
				fmt.Println("Exiting...")
				os.Exit(0)
			}
		}

		m, err := migrate.New(
			"file://database/migrations",
			connString)
		if err != nil {
			fmt.Printf("Error while creating migration instance: %v\n", err)
			os.Exit(1)
		}
		if err := m.Down(); err != nil {
			fmt.Printf("Error while migrating database: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Database migration down done with success. All Tables dropped")
	},
}

func init() {
	migrateCmd.AddCommand(downCmd)
}
