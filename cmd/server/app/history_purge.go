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
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"       // nolint
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/minder/internal/config"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/logger"
)

// historyPurgeCmd represents the `history purge` command
var historyPurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Removes evaluation history entries",
	Long:  `deletes all evaluation history entries older than 30 days, maintaining the latest one per rule/entity pair`,
	RunE:  historyPurgeCommand,
}

func historyPurgeCommand(cmd *cobra.Command, _ []string) error {
	batchSize := viper.GetUint("batch-size")
	dryRun := viper.GetBool("dry-run")

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

	// We maintain up to 30 days of history, plus any record
	// that's the latest for any entity/rule pair.
	threshold := time.Now().UTC().AddDate(0, 0, -30)
	cmd.Printf("Calculated threshold is %s", threshold)

	if err := purgeLoop(ctx, store, threshold, batchSize, dryRun, cmd.Printf); err != nil {
		cliErrorf(cmd, "failed purging evaluation log: %s", err)
	}

	return nil
}

// purgeLoop routine cleans up the evaluation history log by deleting
// all stale records older than a given threshold.
//
// As of the time of this writing, the size of the row structs is 80
// bytes, specifically
//
// * go time is 24 bytes
// * rule id, entity id and evaluation id are 16 bytes UUIDs, adding
// 48 bytes
// * entity type, which is necessary because there's no guarantee that
// the entity id is unique across entity types, is mapped to an
// integer and adds another 16 bytes
//
// Given their size, 4 million records would allocate around 300MB of
// RAM, adding some overhead for the used data structures we estimate
// 500MB total RAM consumption in the worst case.
//
// From the execution time perspective, this is not necessarily the
// best approach, and the time to first byte might become significant
// as usage increases.
func purgeLoop(
	ctx context.Context,
	store db.Store,
	threshold time.Time,
	batchSize uint,
	dryRun bool,
	printf func(format string, a ...any),
) error {
	deleted := 0

	// Note: this command relies on the following statement
	// filtering out records that, despite being older than 30
	// days, are the latest ones for any given entity/rule pair.
	records, err := store.ListEvaluationHistoryStaleRecords(
		ctx,
		db.ListEvaluationHistoryStaleRecordsParams{
			Threshold: threshold,
			Size:      int32(4000000),
		},
	)
	if err != nil {
		return fmt.Errorf("error purging evaluation history: %w", err)
	}

	if len(records) == 0 {
		printf("No records to delete\n")
		return nil
	}

	// Skip deletion if --dry-run was passed.
	if !dryRun {
		deleted, err = deleteEvaluationHistory(
			ctx,
			store,
			records,
			batchSize,
		)
		if err != nil {
			return err
		}
	}

	printf("Done purging history, deleted %d records\n",
		deleted,
	)

	return nil
}

func deleteEvaluationHistory(
	ctx context.Context,
	store db.Store,
	records []db.ListEvaluationHistoryStaleRecordsRow,
	batchSize uint,
) (int, error) {
	deleted := 0
	for {
		if len(records) == 0 {
			break
		}

		// This only happens at the last iteration if the
		// number of records to delete is not a multiple of
		// the batch size.
		if batchSize > uint(len(records)) {
			batchSize = uint(len(records))
		}

		// Deletion is done by evaluation id.
		batch := make([]uuid.UUID, 0, batchSize)
		for _, record := range records[:batchSize] {
			batch = append(batch, record.ID)
		}

		partial, err := db.WithTransaction[int64](store,
			func(qtx db.ExtendQuerier) (int64, error) {
				return qtx.DeleteEvaluationHistoryByIDs(ctx, batch)
			},
		)
		if err != nil {
			return 0, fmt.Errorf("error while deleting old evaluations: %w", err)
		}

		records = records[batchSize:]
		deleted = deleted + int(partial)
	}

	return int(deleted), nil
}

func init() {
	historyCmd.AddCommand(historyPurgeCmd)
	historyPurgeCmd.Flags().UintP("batch-size", "s", 1000, "Size of the deletion batch")
	historyPurgeCmd.Flags().Bool("dry-run", false, "Avoids deleting, printing out details about the operation")
}
