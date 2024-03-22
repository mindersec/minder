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

// Package sql provides the eventer implementation for the SQL database.
package sql

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/v3/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog/log"

	serverconfig "github.com/stacklok/minder/internal/config/server"
)

// BuildPostgreSQLDriver creates a PostgreSQL driver for the eventer
func BuildPostgreSQLDriver(
	ctx context.Context,
	cfg *serverconfig.EventConfig,
) (message.Publisher, message.Subscriber, func(), error) {
	db, _, err := cfg.SQLPubSub.Connection.GetDBConnection(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("unable to connect to events database: %w", err)
	}

	publisher, err := watermillsql.NewPublisher(
		db,
		watermillsql.PublisherConfig{
			SchemaAdapter:        watermillsql.DefaultPostgreSQLSchema{},
			AutoInitializeSchema: true,
		},
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create SQL publisher: %w", err)
	}

	subscriber, err := watermillsql.NewSubscriber(
		db,
		watermillsql.SubscriberConfig{
			SchemaAdapter:    watermillsql.DefaultPostgreSQLSchema{},
			OffsetsAdapter:   watermillsql.DefaultPostgreSQLOffsetsAdapter{},
			InitializeSchema: true,
		},
		watermill.NewStdLogger(false, false),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create SQL subscriber: %w", err)
	}

	return publisher, subscriber, func() {
		err := db.Close()
		if err != nil {
			log.Printf("error closing events database connection: %v", err)
		}
	}, nil
}
