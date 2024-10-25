// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package events provide the eventer object which is responsible for setting up the watermill router
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
	"github.com/open-feature/go-sdk/openfeature"
	promgo "github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"github.com/mindersec/minder/internal/events/common"
	"github.com/mindersec/minder/internal/events/gochannel"
	"github.com/mindersec/minder/internal/events/nats"
	eventersql "github.com/mindersec/minder/internal/events/sql"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/eventer/constants"
	"github.com/mindersec/minder/pkg/eventer/interfaces"
)

// Ensure that the eventer implements the interfaces
var _ interfaces.Publisher = (*eventer)(nil)
var _ interfaces.Service = (*eventer)(nil)
var _ interfaces.Registrar = (*eventer)(nil)
var _ message.Publisher = (*eventer)(nil)

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

type messageInstruments struct {
	// message processing time duration histogram
	messageProcessingTimeHistogram metric.Int64Histogram
}

// NewEventer creates an eventer object which isolates the watermill setup code
func NewEventer(ctx context.Context, _ openfeature.IClient, cfg *serverconfig.EventConfig) (interfaces.Interface, error) {
	if cfg == nil {
		return nil, errors.New("event config is nil")
	}
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
		constants.MetricsNamespace,
		constants.MetricsSubsystem)
	metricsBuilder.AddPrometheusRouterMetrics(router)
	zerolog.Ctx(ctx).Info().Msg("Router Metrics registered")

	meter := otel.Meter("eventer")
	metricInstruments, err := initMetricsInstruments(meter)
	if err != nil {
		return nil, err
	}
	zerolog.Ctx(ctx).Info().Msg("Metrics Instruments registered")

	pub, sub, cl, err := instantiateDriver(ctx, cfg.Driver, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed instantiating driver: %w", err)
	}

	poisonQueueMiddleware, err := middleware.PoisonQueue(pub, constants.DeadLetterQueueTopic)
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
	case constants.GoChannelDriver:
		zerolog.Ctx(ctx).Info().Msg("Using go-channel driver")
		return gochannel.BuildGoChannelDriver(ctx, cfg)
	case constants.SQLDriver:
		zerolog.Ctx(ctx).Info().Msg("Using SQL driver")
		return eventersql.BuildPostgreSQLDriver(ctx, cfg)
	case constants.NATSDriver:
		zerolog.Ctx(ctx).Info().Msg("Using NATS driver")
		return nats.BuildNatsChannelDriver(cfg)
	default:
		zerolog.Ctx(ctx).Info().Msg("Driver unknown")
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
			msg.Metadata.Set(constants.PublishedKey, time.Now().Format(time.RFC3339))
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
func (e *eventer) ConsumeEvents(consumers ...interfaces.Consumer) {
	for _, c := range consumers {
		c.Register(e)
	}
}
