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
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/alexdrl/zerowater"
	"github.com/rs/zerolog"
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
	Register(topic string, handler Handler)

	// HandleAll registers all the consumers with the registrar
	// TODO: should this be a different interface?
	ConsumeEvents(consumers ...Consumer)
}

// Consumer is an interface implemented by components which wish to consume events.
// Once a component has implemented the consumer interface, it can be registered with an
// event router using the HandleAll interface.
type Consumer interface {
	Register(Registrar)
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
}

var _ Registrar = (*Eventer)(nil)
var _ message.Publisher = (*Eventer)(nil)

// Setup creates an Eventer object which isolates the watermill setup code
// TODO: pass in logger
func Setup() (*Eventer, error) {
	l := zerowater.NewZerologLoggerAdapter(
		zerolog.Ctx(context.TODO()).With().Str("component", "watermill").Logger())
	// TODO: parameterize CloseTimeout for testing
	router, err := message.NewRouter(message.RouterConfig{CloseTimeout: time.Second * 10}, l)
	if err != nil {
		return nil, err
	}

	// Router level middleware are executed for every message sent to the router
	router.AddMiddleware(
		// CorrelationID will copy the correlation id from the incoming message's metadata to the produced messages
		middleware.CorrelationID,

		// The handler function is retried if it returns an error.
		// After MaxRetries, the message is Nacked and it's up to the PubSub to resend it.
		middleware.Retry{
			MaxRetries:      3,
			InitialInterval: time.Millisecond * 100,
			Logger:          l,
		}.Middleware,

		// Recoverer handles panics from handlers.
		// In this case, it passes them as errors to the Retry middleware.
		middleware.Recoverer,
	)

	webhpubsub := gochannel.NewGoChannel(gochannel.Config{
		Persistent: true,
	}, l)

	return &Eventer{
		router:            router,
		webhookPublisher:  webhpubsub,
		webhookSubscriber: webhpubsub,
	}, nil
}

// Close closes the router
func (e *Eventer) Close() error {
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
		}
	}

	return e.webhookPublisher.Publish(topic, messages...)
}

// Register subscribes to a topic and handles incoming messages
func (e *Eventer) Register(
	topic string,
	handler message.NoPublishHandlerFunc,
) {
	// From https://stackoverflow.com/questions/7052693/how-to-get-the-name-of-a-function-in-go
	funcName := fmt.Sprintf("%s-%s", runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name(), topic)
	e.router.AddNoPublisherHandler(
		funcName,
		topic,
		e.webhookSubscriber,
		func(msg *message.Message) error {
			if err := handler(msg); err != nil {
				retriable := errors.Is(err, ErrRetriable)
				e.router.Logger().Error("Found error handling message", err, watermill.LogFields{
					"message_uuid": msg.UUID,
					"topic":        topic,
					"handler":      funcName,
					"retriable":    retriable,
				})

				if retriable {
					// if the error is retriable, return it so that the message is retried
					return err
				}
				// otherwise, we've done all we can, so return nil so that the message is acked
				return nil
			}

			return nil
		},
	)
}

// ConsumeEvents allows registration of multiple consumers easily
func (e *Eventer) ConsumeEvents(consumers ...Consumer) {
	for _, c := range consumers {
		c.Register(e)
	}
}
