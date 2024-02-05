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
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"       // nolint
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/authz"
	"github.com/stacklok/minder/internal/config"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/projects"
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

		store := db.NewStore(dbConn)
		return migrateProjectsRootAndMetadata(ctx, cmd, store)
	},
}

func migrateProjectsRootAndMetadata(ctx context.Context, cmd *cobra.Command, store db.Store) error {
	cmd.Println("Migrating projects to have no parent...")
	projs, err := store.ListNonOrgProjects(ctx)
	if err != nil {
		return fmt.Errorf("error while listing non-org projects: %w", err)
	}

	var lastError error
	for _, p := range projs {
		if err := migrateOneProject(ctx, cmd, store, p); err != nil {
			cmd.Printf("Error while migrating project %s: %v\n", p.ID.String(), err)
			lastError = err
		}
	}

	if lastError != nil {
		return lastError
	}

	// Nothing was done with the old org projects, so we can just delete them
	cmd.Println("Deleting old org projects...")
	projs, err = store.ListOldOrgProjects(ctx)
	if err != nil {
		return fmt.Errorf("error while listing old org projects: %w", err)
	}

	for _, p := range projs {
		if err := deleteOrgProject(ctx, store, p); err != nil {
			cmd.Printf("Error while deleting project %s: %v\n", p.ID.String(), err)
			lastError = err
		}
	}

	if lastError != nil {
		return lastError
	}

	return nil
}

func migrateOneProject(ctx context.Context, cmd *cobra.Command, store db.Store, p db.Project) error {
	m := projects.NewSelfEnrolledMetadata()
	if err := json.Unmarshal(p.Metadata, &m); err != nil {
		return fmt.Errorf("error while marshalling old metadata: %w", err)
	}

	newmeta, err := json.Marshal(&m)
	if err != nil {
		return fmt.Errorf("error while marshalling new metadata: %w", err)
	}

	if p.ParentID.Valid {
		// Note that we only migrate projects to be self-contained and don't need to do anything
		// in OpenFGA. This is because the project is already self-contained in OpenFGA.
		cmd.Printf("Migrating project %s to be self-contained\n", p.ID.String())
		if _, err := store.OrphanProject(ctx, db.OrphanProjectParams{
			ID:       p.ID,
			Metadata: newmeta,
		}); err != nil {
			return fmt.Errorf("error while migrating project %s: %w", p.ID.String(), err)
		}
	} else {
		// This is an edge-case where the project is already self-contained, but the metadata
		// is not updated. This is a no-op.
		cmd.Printf("Ensuring project has new metadata")
		if _, err := store.UpdateProjectMeta(ctx, db.UpdateProjectMetaParams{
			ID:       p.ID,
			Metadata: newmeta,
		}); err != nil {
			return fmt.Errorf("error while updating project metadata: %w", err)
		}
	}

	return nil
}

func deleteOrgProject(ctx context.Context, store db.Store, p db.Project) error {
	// GetChildrenProjects actually also returns the project itself, so we need to check
	// if there are more than one project in the list.
	thisAndChildren, err := store.GetChildrenProjects(ctx, p.ID)
	if err != nil {
		return fmt.Errorf("error while getting children projects: %w", err)
	}

	if len(thisAndChildren) > 1 {
		return fmt.Errorf("project %s has children and should not be deleted", p.ID.String())
	}

	_, err = store.DeleteProject(ctx, p.ID)
	if err != nil {
		return fmt.Errorf("error while deleting project: %w", err)
	}

	return nil
}

func init() {
	migrateCmd.AddCommand(upCmd)
}
