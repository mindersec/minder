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
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/stacklok/minder/internal/db"
)

type CancelFunc func()

// GetFakeStore returns a new embedded Postgres database and a cancel function
// to clean up the database.
func GetFakeStore() (db.Store, CancelFunc, error) {
	cfg := embeddedpostgres.DefaultConfig()
	port, err := pickUnusedPort()
	if err != nil {
		return nil, nil, err
	}
	tmpName, err := os.MkdirTemp("", "minder-db-test")
	if err != nil {
		return nil, nil, err
	}
	cleanupDir := func() {
		if err := os.RemoveAll(tmpName); err != nil {
			fmt.Printf("Unable to remove tmpdir %q: %v\n", tmpName, err)
		}
	}
	cfg = cfg.Port(uint32(port)).RuntimePath(tmpName)

	postgres := embeddedpostgres.NewDatabase(cfg)
	if err := postgres.Start(); err != nil {
		return nil, cleanupDir, err
	}
	cancel := func() {
		if err := postgres.Stop(); err != nil {
			fmt.Printf("Unable to stop postgres: %v\n", err)
		}
		cleanupDir()
	}
	sqlDB, err := sql.Open("postgres", cfg.GetConnectionURL()+"?sslmode=disable")

	configpath := "file://../../database/migrations"
	mig, err := migrate.New(configpath, cfg.GetConnectionURL()+"?sslmode=disable")
	if err != nil {
		return nil, cancel, err
	}
	if err := mig.Up(); err != nil {
		return nil, cancel, err
	}

	return db.NewStore(sqlDB), cancel, nil
}

func pickUnusedPort() (int, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
