// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package embedded provides a test-only embedded Postgres database for testing queries.
package embedded

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"sync"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/mindersec/minder/database"
	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/db"
)

// CancelFunc is a function that can be called to clean up resources.
// Pass this to t.Cleanup.
type CancelFunc func()

type sharedPostgres struct {
	postgres *embeddedpostgres.EmbeddedPostgres
	cfg      embeddedpostgres.Config
	inFlight sync.WaitGroup
	lock     sync.Mutex
}

var instance sharedPostgres

// ensurePostgres is a one-shot function to set up an embedded Postgres server.
// Individual tests should use GetFakeStore to get a handle to a unique database
// table space within the shared server.  Using a shared server allows us to
// amortize the cost of starting the server across multiple tests.
func ensurePostgres() (*embeddedpostgres.Config, CancelFunc, error) {
	instance.lock.Lock()
	defer instance.lock.Unlock()

	if instance.postgres != nil {
		return newDBFromShared()
	}

	instance.cfg = embeddedpostgres.DefaultConfig()
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
	instance.cfg = instance.cfg.Port(uint32(port)).RuntimePath(tmpName)

	instance.postgres = embeddedpostgres.NewDatabase(instance.cfg)
	if err := instance.postgres.Start(); err != nil {
		return nil, cleanupDir, fmt.Errorf("unable to start postgres: %w", err)
	}
	cfg, cancel, err := newDBFromShared()
	go func() {
		instance.inFlight.Wait()
		instance.lock.Lock()
		defer instance.lock.Unlock()
		if err := instance.postgres.Stop(); err != nil {
			fmt.Printf("Unable to stop postgres: %v\n", err)
		}
		cleanupDir()
	}()

	return cfg, cancel, err
}

// newDBFromShared is a helper function to create a new database in the shared
// Postgres instance.  It assumes that the instance global is locked.
func newDBFromShared() (*embeddedpostgres.Config, CancelFunc, error) {
	sqlDB, err := sql.Open("postgres", instance.cfg.GetConnectionURL()+"?sslmode=disable")
	if err != nil {
		return nil, nil, fmt.Errorf("unable to open database: %w", err)
	}
	// TODO: make this align with test names in some way.
	dbName, err := crypto.GenerateNonce()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate database name: %w", err)
	}
	instance.inFlight.Add(1)

	_, err = sqlDB.Exec(fmt.Sprintf("CREATE DATABASE %q", dbName))
	if err != nil {
		instance.inFlight.Done()
		return nil, nil, fmt.Errorf("unable to create database: %w", err)
	}
	cfg := instance.cfg
	cfg = cfg.Database(dbName)
	cancel := func() {
		instance.lock.Lock()
		defer instance.lock.Unlock()
		// TODO: do we care about dropping the database?
		instance.inFlight.Done()
	}
	return &cfg, cancel, nil
}

// GetFakeStore returns a new embedded Postgres database and a cancel function
// to clean up the database.
func GetFakeStore() (db.Store, CancelFunc, error) {
	cfg, cancel, err := ensurePostgres()

	if err != nil {
		return nil, cancel, fmt.Errorf("unable to start postgres: %w", err)
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
