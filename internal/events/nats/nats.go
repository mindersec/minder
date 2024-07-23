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

// Package nats provides the eventer implementation for NATS.
package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"

	serverconfig "github.com/stacklok/minder/internal/config/server"
)

// BuildNATSDriver creates a NATS driver for the eventer
func BuildNATSDriver(
	ctx context.Context,
	cfg *serverconfig.EventConfig,
	logger watermill.LoggerAdapter,
) (message.Publisher, message.Subscriber, func(), error) {
	natsCfg := cfg.NATSPubSub

	jetstreamConfig := nats.JetStreamConfig{
		// JetStream is required.
		Disabled:      false,
		AutoProvision: true,
		TrackMsgId:    true,
		AckAsync:      true,
	}

	publisher, err := nats.NewPublisher(
		nats.PublisherConfig{
			URL:       natsCfg.URL,
			JetStream: jetstreamConfig,
		},
		logger,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create NATS publisher: %w", err)
	}

	subscriber, err := nats.NewSubscriber(
		nats.SubscriberConfig{
			URL: natsCfg.URL,
			// TODO: Make these configurable
			CloseTimeout: 30 * time.Second,
			// TODO: Make these configurable
			AckWaitTimeout: 30 * time.Second,
			JetStream:      jetstreamConfig,
		},
		logger,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create NATS subscriber: %w", err)
	}

	return publisher, subscriber, func() {
	}, nil
}
