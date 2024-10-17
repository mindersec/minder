// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reminder

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/v3/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/events/common"
)

func (r *reminder) setupSQLPublisher(ctx context.Context) (message.Publisher, common.DriverCloser, error) {
	logger := zerolog.Ctx(ctx)

	db, _, err := r.cfg.EventConfig.Connection.GetDBConnection(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to connect to events database: %w", err)
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
		return nil, nil, fmt.Errorf("failed to create SQL publisher: %w", err)
	}

	return publisher, func() {
		err := db.Close()
		if err != nil {
			logger.Printf("error closing events database connection: %v", err)
		}
	}, nil
}
