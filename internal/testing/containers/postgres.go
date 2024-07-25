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

package containers

import (
	"context"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// NewPostgresContainer starts a postgres container for testing.
//
// The started instance is configured to accept insecure connections
// on database `minder` with username and password both equal to
// `postgres`.
//
// Along with the usual error, returned values are a connection
// string, a container object that needs to be terminated by the
// caller.
func NewPostgresContainer(ctx context.Context) (string, tc.Container, error) {
	container, err := postgres.Run(ctx,
		"postgres:16.2",
		postgres.WithDatabase("minder"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		tc.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
		WithNopLogger(),
	)
	if err != nil {
		return "", nil, err
	}

	connStr, err := container.ConnectionString(ctx,
		"sslmode=disable",
		"application_name=test",
	)
	if err != nil {
		return "", nil, err
	}

	return connStr, container, nil
}
