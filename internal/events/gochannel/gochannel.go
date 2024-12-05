// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package gochannel provides a gochannel implementation of the eventer
package gochannel

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/alexdrl/zerowater"
	"github.com/rs/zerolog"

	"github.com/mindersec/minder/internal/events/common"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
)

// BuildGoChannelDriver creates a gochannel driver for the eventer
func BuildGoChannelDriver(
	ctx context.Context,
	cfg *serverconfig.EventConfig,
) (message.Publisher, message.Subscriber, common.DriverCloser, error) {
	logger := zerowater.NewZerologLoggerAdapter(zerolog.Ctx(ctx).With().Logger())

	pubsub := gochannel.NewGoChannel(gochannel.Config{
		OutputChannelBuffer: cfg.GoChannel.BufferSize,
		Persistent:          cfg.GoChannel.PersistEvents,
	}, logger)

	return pubsub, pubsub, func() {}, nil
}
