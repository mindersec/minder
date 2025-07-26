// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/mindersec/minder/internal/db"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
)

// This file contains logic shared between different commands.

func wireUpDB(ctx context.Context, cfg *serverconfig.Config) (db.Store, func(), error) {
	zerolog.Ctx(ctx).Debug().
		Str("name", cfg.Database.Name).
		Str("host", cfg.Database.Host).
		Str("user", cfg.Database.User).
		Str("ssl_mode", cfg.Database.SSLMode).
		Int("port", cfg.Database.Port).
		Msg("connecting to minder database")

	dbConn, _, err := cfg.Database.GetDBConnection(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	closer := func() {
		err := dbConn.Close()
		if err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Msg("error closing database connection")
		}
	}

	return db.NewStore(dbConn), closer, nil
}

func confirm(cmd *cobra.Command, message string) bool {
	yes, err := cmd.Flags().GetBool("yes")
	if err != nil {
		// non-fatal error
		cmd.Printf("Error while getting yes flag: %v", err)
	}
	if !yes {
		cmd.Printf("WARNING: %s. Do you want to continue? (y/n): ", message)
		var response string
		_, err := fmt.Scanln(&response)
		if err != nil {
			// for sake of simplicity, exit instead of returning error
			cmd.Printf("error while reading user input: %s", err)
			os.Exit(-1)
		}

		if response != "y" {
			cmd.Printf("Exiting...")
			return false
		}
	}
	return true
}

// cliErrorf prints the error and exits
func cliErrorf(cmd *cobra.Command, message string, args ...any) {
	cmd.Printf(message, args...)
	os.Exit(1)
}
