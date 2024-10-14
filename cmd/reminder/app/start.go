//
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
	"database/sql"
	"fmt"
	"os"
	"os/signal"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/mindersec/minder/internal/config"
	reminderconfig "github.com/mindersec/minder/internal/config/reminder"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/reminder"
	"github.com/mindersec/minder/internal/reminder/logger"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the reminder process",
	Long:  `Start the reminder process to send reminders to the minder server to process entities in background.`,
	RunE:  start,
}

func start(cmd *cobra.Command, _ []string) error {
	ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt)
	defer cancel()

	cfg, err := config.ReadConfigFromViper[reminderconfig.Config](viper.GetViper())
	if err != nil {
		return fmt.Errorf("unable to read config: %w", err)
	}

	err = cfg.Validate()
	if err != nil {
		return fmt.Errorf("error validating config: %w", err)
	}

	ctx = logger.FromFlags(cfg.LoggingConfig).WithContext(ctx)

	dbConn, _, err := cfg.Database.GetDBConnection(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer func(dbConn *sql.DB) {
		err := dbConn.Close()
		if err != nil {
			log.Printf("error closing database connection: %v", err)
		}
	}(dbConn)

	store := db.NewStore(dbConn)
	reminderService, err := reminder.NewReminder(ctx, store, cfg)
	if err != nil {
		return fmt.Errorf("unable to create reminder service: %w", err)
	}
	defer reminderService.Stop()

	errg, ctx := errgroup.WithContext(ctx)

	errg.Go(func() error {
		return reminderService.Start(ctx)
	})

	return errg.Wait()
}

func init() {
	RootCmd.AddCommand(startCmd)
}
