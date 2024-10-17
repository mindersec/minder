// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package app provides the entrypoint for the minder migrations
package app

import (
	"context"
	"errors"
	"fmt"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"       // nolint
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/config"
	serverconfig "github.com/mindersec/minder/internal/config/server"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/logger"
)

// purgeCmd represents the `encryption purge-sessions` command
var purgeCmd = &cobra.Command{
	Use:   "purge-sessions",
	Short: "Purge stale session states",
	Long:  `deletes all session states which are more than 24 hours old`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.ReadConfigFromViper[serverconfig.Config](viper.GetViper())
		if err != nil {
			cliErrorf(cmd, "unable to read config: %s", err)
		}

		ctx := logger.FromFlags(cfg.LoggingConfig).WithContext(context.Background())

		// instantiate `db.Store` so we can run queries
		store, closer, err := wireUpDB(ctx, cfg)
		if err != nil {
			cliErrorf(cmd, "unable to connect to database: %s", err)
		}
		defer closer()

		yes := confirm(cmd, "Running this command will purge stale sessions")
		if !yes {
			return nil
		}

		// Clean up old session states (along with their secrets)
		sessionsDeleted, err := deleteStaleSessions(ctx, cmd, store)
		if err != nil {
			// if we cancel or have nothing to migrate...
			if errors.Is(err, errCancelRotation) {
				cmd.Printf("Cleanup canceled, exiting\n")
				return nil
			}
			cliErrorf(cmd, "error while deleting stale sessions: %s", err)
		}
		if sessionsDeleted != 0 {
			cmd.Printf("Successfully deleted %d stale sessions\n", sessionsDeleted)
		}

		return nil
	},
}

func deleteStaleSessions(
	ctx context.Context,
	cmd *cobra.Command,
	store db.Store,
) (int64, error) {
	return db.WithTransaction[int64](store, func(qtx db.ExtendQuerier) (int64, error) {
		// delete any sessions more than one day old
		deleted, err := qtx.DeleteExpiredSessionStates(ctx)
		if err != nil {
			return 0, err
		}

		// skip the confirmation if there's nothing to do
		if deleted == 0 {
			cmd.Printf("No stale sessions to delete\n")
			return 0, nil
		}

		// one last chance to reconsider your choices
		yes := confirm(cmd, fmt.Sprintf("About to delete %d stale sessions", deleted))
		if !yes {
			return 0, errCancelRotation
		}
		return deleted, nil
	})
}

func init() {
	encryptionCmd.AddCommand(purgeCmd)
}
