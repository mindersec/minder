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
	natsinternal "github.com/mindersec/minder/internal/events/nats"
	"github.com/mindersec/minder/pkg/eventer/constants"
)

func (r *reminder) getMessagePublisher(ctx context.Context) (message.Publisher, common.DriverCloser, error) {
	switch r.cfg.EventConfig.Driver {
	case constants.NATSDriver:
		return r.setupNATSPublisher(ctx)
	case constants.SQLDriver:
		return r.setupSQLPublisher(ctx)
	default:
		return nil, nil, fmt.Errorf("unknown publisher type: %s", r.cfg.EventConfig.Driver)
	}
}

func (r *reminder) setupNATSPublisher(_ context.Context) (message.Publisher, common.DriverCloser, error) {
	pub, _, cl, err := natsinternal.BuildNatsChannelDriver(&r.cfg.EventConfig.NatsConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create NATS publisher: %w", err)
	}
	return pub, cl, nil
}

func (r *reminder) setupSQLPublisher(ctx context.Context) (message.Publisher, common.DriverCloser, error) {
	logger := zerolog.Ctx(ctx)

	db, _, err := r.cfg.EventConfig.SQLPubConfig.Connection.GetDBConnection(ctx)
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
