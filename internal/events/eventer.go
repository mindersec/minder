//
// Copyright 2023 Stacklok, Inc.
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

// Package events provides the eventer object which is responsible for setting up the watermill router
// and handling the incoming events
package events

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/metrics"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/alexdrl/zerowater"
	promgo "github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/events/common"
	gochannel "github.com/stacklok/minder/internal/events/gochannel"
	eventersql "github.com/stacklok/minder/internal/events/sql"
)

// eventer is a wrapper over the relevant eventing objects in such
// a way that they can be easily accessible and configurable.
type eventer struct {
	router *message.Router
	// webhookPublisher will gather events coming into the webhook and publish them
	webhookPublisher message.Publisher
	// webhookSubscriber will subscribe to the webhook topic and handle incoming events
	webhookSubscriber message.Subscriber
	// TODO: We'll have a Final publisher that will publish to the final topic
	msgInstruments *messageInstruments

	closer common.DriverCloser
}

var _ Publisher = (*eventer)(nil)
var _ Service = (*eventer)(nil)

type messageInstruments struct {
	// message processing time duration histogram
	messageProcessingTimeHistogram metric.Int64Histogram
}

var _ Registrar = (*eventer)(nil)
var _ message.Publisher = (*eventer)(nil)

// Setup creates an eventer object which isolates the watermill setup code
func Setup(ctx context.Context, cfg *serverconfig.EventConfig) (Interface, error) {
	if cfg == nil {
		return nil, errors.New("event config is nil")
	}

	l := zerowater.NewZerologLoggerAdapter(
		zerolog.Ctx(ctx).With().Str("component", "watermill").Logger())

	router, err := message.NewRouter(message.RouterConfig{
		CloseTimeout: time.Duration(cfg.RouterCloseTimeout) * time.Second,
	}, l)
	if err != nil {
		return nil, err
	}

	metricsBuilder := metrics.NewPrometheusMetricsBuilder(
		promgo.DefaultRegisterer,
		metricsNamespace,
		metricsSubsystem)
	metricsBuilder.AddPrometheusRouterMetrics(router)

	meter := otel.Meter("eventer")
	metricInstruments, err := initMetricsInstruments(meter)
	if err != nil {
		return nil, err
	}

	pub, sub, cl, err := instantiateDriver(ctx, cfg.Driver, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed instantiating driver: %w", err)
	}

	poisonQueueMiddleware, err := middleware.PoisonQueue(pub, DeadLetterQueueTopic)
	if err != nil {
		return nil, fmt.Errorf("failed instantiating poison queue: %w", err)
	}
	// Router level middleware are executed for every message sent to the router
	router.AddMiddleware(
		recordMetrics(metricInstruments),
		poisonQueueMiddleware,
		middleware.Retry{
			MaxRetries:      3,
			InitialInterval: time.Millisecond * 100,
			Logger:          l,
		}.Middleware,
		// CorrelationID will copy the correlation id from the incoming message's metadata to the produced messages
		middleware.CorrelationID,
	)

	pubWithMetrics, err := metricsBuilder.DecoratePublisher(pub)
	if err != nil {
		return nil, fmt.Errorf("failed to decorate publisher: %w", err)
	}

	subWithMetrics, err := metricsBuilder.DecorateSubscriber(sub)
	if err != nil {
		return nil, fmt.Errorf("failed to decorate subscriber: %w", err)
	}

	return &eventer{
		router:            router,
		webhookPublisher:  pubWithMetrics,
		webhookSubscriber: subWithMetrics,
		closer: func() {
			//nolint:gosec // It's fine if there's an error as long as we close the router
			pubWithMetrics.Close()
			//nolint:gosec // It's fine if there's an error as long as we close the router
			subWithMetrics.Close()
			// driver close
			cl()
		},
		msgInstruments: metricInstruments,
	}, nil
}

func instantiateDriver(
	ctx context.Context,
	driver string,
	cfg *serverconfig.EventConfig,
) (message.Publisher, message.Subscriber, common.DriverCloser, error) {
	switch driver {
	case GoChannelDriver:
		return gochannel.BuildGoChannelDriver(cfg)
	case SQLDriver:
		return eventersql.BuildPostgreSQLDriver(ctx, cfg)
	default:
		return nil, nil, nil, fmt.Errorf("unknown driver %s", driver)
	}
}

// Close closes the router
func (e *eventer) Close() error {
	e.closer()
	return e.router.Close()
}

// Run runs the router, blocks until the router is closed
func (e *eventer) Run(ctx context.Context) error {
	return e.router.Run(ctx)
}

// Running returns a channel which allows you to wait until the
// event router has started.
func (e *eventer) Running() chan struct{} {
	return e.router.Running()
}

// Publish implements message.Publisher
func (e *eventer) Publish(topic string, messages ...*message.Message) error {
	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)

	if ok && details != nil {
		for idx := range messages {
			msg := messages[idx]
			e.router.Logger().Debug("Publishing message", watermill.LogFields{
				"message_uuid": msg.UUID,
				"topic":        topic,
				"handler":      details.Name(),
				"component":    "eventer",
				"function":     "Publish",
			})
			msg.Metadata.Set(PublishedKey, time.Now().Format(time.RFC3339))
		}
	}

	return e.webhookPublisher.Publish(topic, messages...)
}

// Register subscribes to a topic and handles incoming messages
func (e *eventer) Register(
	topic string,
	handler message.NoPublishHandlerFunc,
	mdw ...message.HandlerMiddleware,
) {
	// From https://stackoverflow.com/questions/7052693/how-to-get-the-name-of-a-function-in-go
	funcName := fmt.Sprintf("%s-%s", runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name(), topic)
	hand := e.router.AddNoPublisherHandler(
		funcName,
		topic,
		e.webhookSubscriber,
		func(msg *message.Message) error {
			if err := handler(msg); err != nil {
				e.router.Logger().Error("Found error handling message", err, watermill.LogFields{
					"message_uuid": msg.UUID,
					"topic":        topic,
					"handler":      funcName,
					"component":    "eventer",
				})

				return err
			}

			e.router.Logger().Info("Handled message", watermill.LogFields{
				"message_uuid": msg.UUID,
				"topic":        topic,
				"handler":      funcName,
				"component":    "eventer",
			})

			return nil
		},
	)

	for _, m := range mdw {
		hand.AddMiddleware(m)
	}
}

// ConsumeEvents allows registration of multiple consumers easily
func (e *eventer) ConsumeEvents(consumers ...Consumer) {
	for _, c := range consumers {
		c.Register(e)
	}
}
