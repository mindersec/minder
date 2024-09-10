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
	"database/sql"
	"fmt"
	"net"
	"os"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/stacklok/minder/database"
	"github.com/stacklok/minder/internal/db"
)

// CancelFunc is a function that can be called to clean up resources.
// Pass this to t.Cleanup.
type CancelFunc func()

// GetFakeStore returns a new embedded Postgres database and a cancel function
// to clean up the database.
func GetFakeStore() (db.Store, CancelFunc, error) {
	cfg := embeddedpostgres.DefaultConfig()
	port, err := pickUnusedPort()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to pick a port: %w", err)
	}
	tmpName, err := os.MkdirTemp("", "minder-db-test")
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create tmpdir: %w", err)
	}
	cleanupDir := func() {
		if err := os.RemoveAll(tmpName); err != nil {
			fmt.Printf("Unable to remove tmpdir %q: %v\n", tmpName, err)
		}
	}
	cfg = cfg.Port(uint32(port)).RuntimePath(tmpName)

	postgres := embeddedpostgres.NewDatabase(cfg)
	if err := postgres.Start(); err != nil {
		return nil, cleanupDir, fmt.Errorf("unable to start postgres: %w", err)
	}
	cancel := func() {
		if err := postgres.Stop(); err != nil {
			fmt.Printf("Unable to stop postgres: %v\n", err)
		}
		cleanupDir()
	}
	sqlDB, err := sql.Open("postgres", cfg.GetConnectionURL()+"?sslmode=disable")
	if err != nil {
		return nil, cancel, fmt.Errorf("unable to open database: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, cancel, fmt.Errorf("unable to ping database: %w", err)
	}

	mig, err := database.NewFromConnectionString(cfg.GetConnectionURL() + "?sslmode=disable")
	if err != nil {
		return nil, cancel, fmt.Errorf("unable to create migration: %w", err)
	}
	if err := mig.Up(); err != nil {
		return nil, cancel, fmt.Errorf("unable to run migration: %w", err)
	}

	return db.NewStore(sqlDB), cancel, nil
}

func pickUnusedPort() (uint32, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	// largest TCP port is 2^16, overflow should not happen
	port := l.Addr().(*net.TCPAddr).Port
	if port < 0 {
		return 0, fmt.Errorf("invalid port %d", port)
	}
	// nolint: gosec
	return uint32(port), nil
}
