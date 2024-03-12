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

// Package database provides the database migration tooling for the minder application.
package database

import (
	"embed"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var fs embed.FS

// migrationsFromSource returns a migration source driver from the embedded migrations.
func migrationsFromSource() source.Driver {
	d, err := iofs.New(fs, "migrations")
	if err != nil {
		panic(err)
	}

	return d
}

// Migrator is the interface for the migration tooling.
type Migrator interface {
	Up() error
	Down() error
	Steps(int) error
	Version() (uint, bool, error)
}

// NewFromConnectionString returns a new migration instance from the given connection string.
func NewFromConnectionString(connString string) (Migrator, error) {
	d := migrationsFromSource()
	return migrate.NewWithSourceInstance("iofs", d, connString)
}
