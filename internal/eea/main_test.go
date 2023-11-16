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

package eea_test

import (
	"database/sql"
	"log"
	"os"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // nolint
	_ "github.com/golang-migrate/migrate/v4/source/file"       // nolint
	_ "github.com/lib/pq"

	"github.com/stacklok/minder/internal/db"
)

var testQueries db.Store
var testDB *sql.DB

func TestMain(m *testing.M) {
	os.Exit(runTestWithInProcessPostgres(m))
}

func runTestWithInProcessPostgres(m *testing.M) int {
	tmpName, err := os.MkdirTemp("", "mediator-db-test")
	if err != nil {
		log.Println("cannot create tmpdir:", err)
		return -1
	}

	defer func() {
		if err := os.RemoveAll(tmpName); err != nil {
			log.Println("cannot remove tmpdir:", err)
		}
	}()

	dbCfg := embeddedpostgres.DefaultConfig().
		Database("mediator").
		RuntimePath(tmpName).
		Port(5434)
	postgres := embeddedpostgres.NewDatabase(dbCfg)

	if err := postgres.Start(); err != nil {
		log.Println("cannot start postgres:", err)
		return -1
	}
	defer func() {
		if err := postgres.Stop(); err != nil {
			log.Println("cannot stop postgres:", err)
		}
	}()

	testDB, err = sql.Open("postgres", "user=postgres dbname=mediator password=postgres host=localhost port=5434 sslmode=disable")
	if err != nil {
		log.Println("cannot connect to db test instance:", err)
		return -1
	}

	configPath := "file://../../database/migrations"
	mig, err := migrate.New(configPath, dbCfg.GetConnectionURL()+"?sslmode=disable")
	if err != nil {
		log.Printf("Error while creating migration instance (%s): %v\n", configPath, err)
		return -1
	}

	if err := mig.Up(); err != nil {
		log.Println("cannot run db migrations:", err)
		return -1
	}

	defer func() {
		if err := testDB.Close(); err != nil {
			log.Println("cannot close test db:", err)
		}
	}()

	testQueries = db.NewStore(testDB)

	// Run tests
	return m.Run()
}
