// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"database/sql"
	"os"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"       // nolint
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

const (
	// UseExternalDBEnvVar is the environment variable that, when set, will
	// enable using an external postgres database instead of the in-process one.
	// Useful for debugging.
	UseExternalDBEnvVar = "MINDER_TEST_EXTERNAL_DB"
)

var testQueries *Queries
var testDB *sql.DB

func TestMain(m *testing.M) {
	var runDBTests = runTestWithInProcessPostgres
	if useExternalDB() {
		log.Print("Using external database for tests")
		runDBTests = runTestWithExternalPostgres
	}
	os.Exit(runDBTests(m))
}

func runTestWithExternalPostgres(m *testing.M) int {
	connStr := "user=postgres dbname=minder password=postgres host=localhost sslmode=disable"

	testDB, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to db test instance")
	}

	testQueries = New(testDB)

	return m.Run()
}

func runTestWithInProcessPostgres(m *testing.M) int {
	tmpName, err := os.MkdirTemp("", "minder-db-test")
	if err != nil {
		log.Err(err).Msg("cannot create tmpdir")
		return -1
	}

	defer func() {
		if err := os.RemoveAll(tmpName); err != nil {
			log.Err(err).Msg("cannot remove tmpdir")
		}
	}()

	dbCfg := embeddedpostgres.DefaultConfig().
		Database("minder").
		RuntimePath(tmpName).
		Port(5433)
	postgres := embeddedpostgres.NewDatabase(dbCfg)

	if err := postgres.Start(); err != nil {
		log.Err(err).Msg("cannot start postgres")
		return -1
	}
	defer func() {
		if err := postgres.Stop(); err != nil {
			log.Err(err).Msg("cannot stop postgres")
		}
	}()

	testDB, err = sql.Open("postgres", "user=postgres dbname=minder password=postgres host=localhost port=5433 sslmode=disable")
	if err != nil {
		log.Err(err).Msg("cannot connect to db test instance")
		return -1
	}

	configPath := "file://../../database/migrations"
	mig, err := migrate.New(configPath, dbCfg.GetConnectionURL()+"?sslmode=disable")
	if err != nil {
		log.Printf("Error while creating migration instance (%s): %v\n", configPath, err)
		return -1
	}

	if err := mig.Up(); err != nil {
		log.Err(err).Msg("cannot run db migrations")
		return -1
	}

	defer func() {
		if err := testDB.Close(); err != nil {
			log.Err(err).Msg("cannot close test db")
		}
	}()

	testQueries = New(testDB)

	// Run tests
	return m.Run()
}

func useExternalDB() bool {
	_, ok := os.LookupEnv(UseExternalDBEnvVar)
	return ok
}
