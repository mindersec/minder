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

package reminder

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/v3/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/events/common"
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
