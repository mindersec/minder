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

package db

import (
	"database/sql"
	"os"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/rs/zerolog/log"
	"github.com/signalfx/splunk-otel-go/instrumentation/database/sql/splunksql"
	_ "github.com/signalfx/splunk-otel-go/instrumentation/github.com/lib/pq/splunkpq"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestOtelPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf(`
The code did panic. This usually means that some OTEL dependency
introduced a breaking change or regression that is only detected at
run time.
`)
		}
	}()

	tmpName, err := os.MkdirTemp("", "minder-db-test")
	require.NoError(t, err)
	defer func() {
		if err := os.RemoveAll(tmpName); err != nil {
			log.Err(err).Msg("cannot remove tmpdir")
		}
	}()

	dbCfg := embeddedpostgres.DefaultConfig().
		Database("minder").
		RuntimePath(tmpName).
		Port(5434)
	postgres := embeddedpostgres.NewDatabase(dbCfg)

	err = postgres.Start()
	require.NoError(t, err)
	defer func() {
		if err := postgres.Stop(); err != nil {
			log.Err(err).Msg("cannot stop postgres")
		}
	}()

	conn1 := connect(t, "postgres", "user=postgres dbname=minder password=postgres host=localhost port=5434 sslmode=disable")
	require.NotNil(t, conn1)
	conn2 := connect(t, "postgres", "user=postgres dbname=minder password=postgres host=localhost port=5434 sslmode=disable")
	require.NotNil(t, conn2)

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
	)
	otel.SetMeterProvider(mp)
}

func connect(t *testing.T, driver string, connStr string) *sql.DB {
	t.Helper()

	conn, err := splunksql.Open(driver, connStr)
	require.NotNil(t, conn)
	require.NoError(t, err)

	_, err = conn.Exec("SELECT 1")
	require.NoError(t, err)

	return conn
}
