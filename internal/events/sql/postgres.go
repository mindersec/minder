// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package sql provides the eventer implementation for the SQL database.
package sql

import (
	"context"
	"fmt"

	watermillsql "github.com/ThreeDotsLabs/watermill-sql/v3/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/alexdrl/zerowater"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	serverconfig "github.com/mindersec/minder/pkg/config/server"
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

	logger := zerowater.NewZerologLoggerAdapter(zerolog.Ctx(ctx).With().Logger())

	publisher, err := watermillsql.NewPublisher(
		db,
		watermillsql.PublisherConfig{
			SchemaAdapter:        watermillsql.DefaultPostgreSQLSchema{},
			AutoInitializeSchema: true,
		},
		logger,
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
			AckDeadline:      &cfg.SQLPubSub.AckDeadline,
		},
		logger,
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
