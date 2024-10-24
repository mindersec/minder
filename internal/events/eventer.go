// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package events provides the eventer object which is responsible for setting up the watermill router
// and handling the incoming events
package events

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/events/common"
	"github.com/mindersec/minder/internal/events/gochannel"
	"github.com/mindersec/minder/internal/events/nats"
	eventersql "github.com/mindersec/minder/internal/events/sql"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
)

// InstantiateDriver creates a new driver based on the driver string
func InstantiateDriver(
	ctx context.Context,
	driver string,
	cfg *serverconfig.EventConfig,
) (message.Publisher, message.Subscriber, common.DriverCloser, error) {
	switch driver {
	case GoChannelDriver:
		zerolog.Ctx(ctx).Info().Msg("Using go-channel driver")
		return gochannel.BuildGoChannelDriver(ctx, cfg)
	case SQLDriver:
		zerolog.Ctx(ctx).Info().Msg("Using SQL driver")
		return eventersql.BuildPostgreSQLDriver(ctx, cfg)
	case NATSDriver:
		zerolog.Ctx(ctx).Info().Msg("Using NATS driver")
		return nats.BuildNatsChannelDriver(cfg)
	default:
		zerolog.Ctx(ctx).Info().Msg("Driver unknown")
		return nil, nil, nil, fmt.Errorf("unknown driver %s", driver)
	}
}
