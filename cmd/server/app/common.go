// Copyright 2024 Stacklok, Inc.
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

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
)

// This file contains logic shared between different commands.

func wireUpDB(ctx context.Context, cfg *serverconfig.Config) (db.Store, func(), error) {
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

// cliError prints the error and exits
func cliErrorf(cmd *cobra.Command, message string, args ...any) {
	cmd.Printf(message, args...)
	os.Exit(1)
}
