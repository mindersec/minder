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
	watermillsql "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/metrics"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/alexdrl/zerowater"
	promgo "github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	serverconfig "github.com/stacklok/minder/internal/config/server"
)

// Metadata added to Messages
const (
	ProviderDeliveryIdKey     = "id"
	ProviderTypeKey           = "provider"
	ProviderSourceKey         = "source"
	GithubWebhookEventTypeKey = "type"

	GoChannelDriver = "go-channel"
	SQLDriver       = "sql"

	DeadLetterQueueTopic = "dead_letter_queue"
	PublishedKey         = "published_at"
)

const (
	metricsNamespace = "minder"
	metricsSubsystem = "eventer"
)

const (
	// ExecuteEntityEventTopic is the topic for internal webhook events
	ExecuteEntityEventTopic = "execute.entity.event"
	// FlushEntityEventTopic is the topic for flushing internal webhook events
	FlushEntityEventTopic = "flush.entity.event"
)

// Handler is an alias for the watermill handler type, which is both wordy and may be
// detail we don't want to expose.
type Handler = message.NoPublishHandlerFunc

// Registrar provides an interface which allows an event router to expose
// itself to event consumers.
type Registrar interface {
	// Register requests that the message router calls handler for each message on topic.
	// It is valid to call Register multiple times with the same topic and different handler
	// functions, or to call Register multiple times with different topics and the same
	// handler function.  It's allowed to call Register with both argument the same, but
	// then events will be delivered twice to the handler, which is probably not what you want.
	Register(topic string, handler Handler, mdw ...message.HandlerMiddleware)
}

// Consumer is an interface implemented by components which wish to consume events.
// Once a component has implemented the consumer interface, it can be registered with an
// event router using the HandleAll interface.
type Consumer interface {
	Register(Registrar)
}

// AggregatorMiddleware is an interface that allows the eventer to
// add middleware to the router
type AggregatorMiddleware interface {
	AggregateMiddleware(h message.HandlerFunc) message.HandlerFunc
}

type driverCloser func()

// Interface provides an abstract interface over Eventer (and mocks)
type Interface interface {
	// Publish implements message.Publisher
	Publish(topic string, messages ...*message.Message) error

	// Register subscribes to a topic and handles incoming messages
	Register(topic string, handler message.NoPublishHandlerFunc, mdw ...message.HandlerMiddleware)

	// ConsumeEvents allows registration of multiple consumers easily
	ConsumeEvents(consumers ...Consumer)
	// Close closes the router
	Close() error
	// Run runs the router, blocks until the router is closed
	Run(ctx context.Context) error

	// Running returns a channel which allows you to wait until the
	// event router has started.
	Running() chan struct{}
}

// Eventer is a wrapper over the relevant eventing objects in such
// a way that they can be easily accessible and configurable.
type Eventer struct {
	router *message.Router
	// webhookPublisher will gather events coming into the webhook and publish them
	webhookPublisher message.Publisher
	// webhookSubscriber will subscribe to the webhook topic and handle incoming events
	webhookSubscriber message.Subscriber
	// TODO: We'll have a Final publisher that will publish to the final topic
	msgInstruments *messageInstruments

	closer driverCloser
}

var _ Interface = (*Eventer)(nil)

type messageInstruments struct {
	// message processing time duration histogram
	messageProcessingTimeHistogram metric.Int64Histogram
}

var _ Registrar = (*Eventer)(nil)
var _ message.Publisher = (*Eventer)(nil)

// Setup creates an Eventer object which isolates the watermill setup code
func Setup(ctx context.Context, cfg *serverconfig.EventConfig) (*Eventer, error) {
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

	return &Eventer{
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

func recordMetrics(instruments *messageInstruments) func(h message.HandlerFunc) message.HandlerFunc {
	metricsFunc := func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			var processingTime time.Duration
			if publishedAt := msg.Metadata.Get(PublishedKey); publishedAt != "" {
				if parsedTime, err := time.Parse(time.RFC3339, publishedAt); err == nil {
					processingTime = time.Since(parsedTime)
				}
			}

			res, err := h(msg)

			// Defer the DLQ tracking logic to after the message has been processed by other middlewares,
			// including the deferred PoisonQueue middleware functionality,
			// so that we can check if it has been poisoned or not.
			isPoisoned := msg.Metadata.Get(middleware.ReasonForPoisonedKey) != ""
			instruments.messageProcessingTimeHistogram.Record(
				msg.Context(),
				processingTime.Milliseconds(),
				metric.WithAttributes(attribute.Bool("poison", isPoisoned)),
			)

			return res, err
		}
	}
	return metricsFunc
}

func instantiateDriver(
	ctx context.Context,
	driver string,
	cfg *serverconfig.EventConfig,
) (message.Publisher, message.Subscriber, driverCloser, error) {
	switch driver {
	case GoChannelDriver:
		return buildGoChannelDriver(cfg)
	case SQLDriver:
		return buildPostgreSQLDriver(ctx, cfg)
	default:
		return nil, nil, nil, fmt.Errorf("unknown driver %s", driver)
	}
}

func buildGoChannelDriver(cfg *serverconfig.EventConfig) (message.Publisher, message.Subscriber, driverCloser, error) {
	pubsub := gochannel.NewGoChannel(gochannel.Config{
		OutputChannelBuffer: cfg.GoChannel.BufferSize,
		Persistent:          cfg.GoChannel.PersistEvents,
	}, nil)

	return pubsub, pubsub, func() {}, nil
}

func buildPostgreSQLDriver(
	ctx context.Context,
	cfg *serverconfig.EventConfig,
) (message.Publisher, message.Subscriber, driverCloser, error) {
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

// Close closes the router
func (e *Eventer) Close() error {
	e.closer()
	return e.router.Close()
}

// Run runs the router, blocks until the router is closed
func (e *Eventer) Run(ctx context.Context) error {
	return e.router.Run(ctx)
}

// Running returns a channel which allows you to wait until the
// event router has started.
func (e *Eventer) Running() chan struct{} {
	return e.router.Running()
}

// Publish implements message.Publisher
func (e *Eventer) Publish(topic string, messages ...*message.Message) error {
	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)

	if ok && details != nil {
		for idx := range messages {
			msg := messages[idx]
			// TODO: This should probably be debugging info
			e.router.Logger().Info("Publishing messages", watermill.LogFields{
				"message_uuid": msg.UUID,
				"topic":        topic,
				"handler":      details.Name(),
			})
			msg.Metadata.Set(PublishedKey, time.Now().Format(time.RFC3339))
		}
	}

	return e.webhookPublisher.Publish(topic, messages...)
}

// Register subscribes to a topic and handles incoming messages
func (e *Eventer) Register(
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
				})

				return err
			}

			e.router.Logger().Info("Handled message", watermill.LogFields{
				"message_uuid": msg.UUID,
				"topic":        topic,
				"handler":      funcName,
			})

			return nil
		},
	)

	for _, m := range mdw {
		hand.AddMiddleware(m)
	}
}

// ConsumeEvents allows registration of multiple consumers easily
func (e *Eventer) ConsumeEvents(consumers ...Consumer) {
	for _, c := range consumers {
		c.Register(e)
	}
}

func initMetricsInstruments(meter metric.Meter) (*messageInstruments, error) {
	histogram, err := createProcessingLatencyHistogram(meter)
	if err != nil {
		return nil, err
	}

	return &messageInstruments{
		messageProcessingTimeHistogram: histogram,
	}, nil
}

func createProcessingLatencyHistogram(meter metric.Meter) (metric.Int64Histogram, error) {
	processingLatencyHistogram, err := meter.Int64Histogram("messages.processing_delay",
		metric.WithDescription("Duration between a message being enqueued and dequeued for processing"),
		metric.WithUnit("ms"),
		// Pick a set of bucket boundaries that span out to 10 minutes
		metric.WithExplicitBucketBoundaries(0, 500, 1000, 2000, 5000, 10000, 30000, 60000, 120000, 300000, 600000),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create message processing processingLatencyHistogram: %w", err)
	}
	return processingLatencyHistogram, nil
}
