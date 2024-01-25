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
	"github.com/stacklok/minder/internal/db"
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
			storeID, err := ensureAuthzStore(ctx, cmd, authzw)
			if err != nil {
				return err
			}

			authzw.GetClient().SetStoreId(storeID)
		}

		mID, err := authzw.WriteModel(ctx)
		if err != nil {
			return fmt.Errorf("error while writing authz model: %w", err)
		}

		if err := authzw.GetClient().SetAuthorizationModelId(mID); err != nil {
			return fmt.Errorf("error setting authz model ID: %w", err)
		}

		store := db.NewStore(dbConn)
		if err := migratePermsToFGA(ctx, store, authzw, cmd); err != nil {
			return fmt.Errorf("error while migrating permissions to FGA: %w", err)
		}

		cmd.Printf("Wrote authz model %s to store.\n", mID)

		return nil
	},
}

func ensureAuthzStore(ctx context.Context, cmd *cobra.Command, authzw *authz.ClientWrapper) (string, error) {
	storeName := authzw.GetConfig().StoreName
	storeID, err := authzw.FindStoreByName(ctx)
	if err != nil && !errors.Is(err, authz.ErrStoreNotFound) {
		return "", err
	} else if errors.Is(err, authz.ErrStoreNotFound) {
		cmd.Printf("Creating authz store %s\n", storeName)
		id, err := authzw.CreateStore(ctx)
		if err != nil {
			return "", err
		}
		cmd.Printf("Created authz store %s/%s\n", id, storeName)
		return id, nil
	}

	cmd.Printf("Not creating store. Found store with name '%s' and ID '%s'.\n",
		storeName, storeID)

	return storeID, nil
}

func migratePermsToFGA(ctx context.Context, store db.Store, authzw *authz.ClientWrapper, cmd *cobra.Command) error {
	cmd.Println("Migrating permissions to FGA...")

	var i int32 = 0
	for {
		userList, err := store.ListUsers(ctx, db.ListUsersParams{Limit: 100, Offset: i})
		if err != nil {
			return fmt.Errorf("error while listing users: %w", err)
		}
		i = i + 100
		cmd.Printf("Found %d users to migrate\n", len(userList))
		if len(userList) == 0 {
			break
		}

		for _, user := range userList {
			projs, err := store.GetUserProjects(ctx, user.ID)
			if err != nil {
				cmd.Printf("Skipping user %d since getting user projects yielded error: %s\n",
					user.ID, err)
				continue
			}

			for _, proj := range projs {
				cmd.Printf("Migrating user to FGA for project %s\n", proj.ProjectID)
				if err := authzw.Write(
					ctx, user.IdentitySubject, authz.AuthzRoleAdmin, proj.ProjectID,
				); err != nil {
					cmd.Printf("Error while writing permission for user %d: %s\n", user.ID, err)
					continue
				}
			}
		}
	}

	cmd.Println("Done migrating permissions to FGA")

	return nil
}

func init() {
	migrateCmd.AddCommand(upCmd)
}
