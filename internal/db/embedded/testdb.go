// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package embedded provides a test-only embedded Postgres database for testing queries.
package embedded

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/stacklok/minder/database"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/testing/containers"
)

// CancelFunc is a function that can be called to clean up resources.
// Pass this to t.Cleanup.
type CancelFunc func()

// GetFakeStore returns a new embedded Postgres database and a cancel function
// to clean up the database.
func GetFakeStore() (db.Store, CancelFunc, error) {
	ctx := context.Background()
	connStr, postgres, err := containers.NewPostgresContainer(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to start postgres: %w", err)
	}
	cancel := func() {
		if err := postgres.Terminate(ctx); err != nil {
			fmt.Printf("Unable to stop postgres: %v\n", err)
		}
	}

	sqlDB, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, cancel, fmt.Errorf("unable to open database: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, cancel, fmt.Errorf("unable to ping database: %w", err)
	}

	mig, err := database.NewFromConnectionString(connStr)
	if err != nil {
		return nil, cancel, fmt.Errorf("unable to create migration: %w", err)
	}
	if err := mig.Up(); err != nil {
		return nil, cancel, fmt.Errorf("unable to run migration: %w", err)
	}

	return db.NewStore(sqlDB), cancel, nil
}
