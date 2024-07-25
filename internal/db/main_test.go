//
// Copyright 2023 Stacklok, Inc.
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

package db

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"       // nolint
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"

	"github.com/stacklok/minder/internal/testing/containers"
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
	var runDBTests = runTestWithPostgresContainer
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

func runTestWithPostgresContainer(m *testing.M) int {
	ctx := context.Background()
	connStr, postgres, err := containers.NewPostgresContainer(ctx)
	if err != nil {
		log.Err(err).Msg("cannot start postgres container")
		return -1
	}
	defer func() {
		if err := postgres.Terminate(ctx); err != nil {
			log.Err(err).Msg("cannot stop postgres container")
		}
	}()

	testDB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Err(err).Msg("cannot connect to db test instance")
		return -1
	}

	configPath := "file://../../database/migrations"
	mig, err := migrate.New(configPath, connStr)
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
