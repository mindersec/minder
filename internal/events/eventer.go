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
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/message/router/plugin"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
)

// Eventer is a wrapper over the relevant eventing objects in such
// a way that they can be easily accessible and configurable.
type Eventer struct {
	Router *message.Router
	// WebhookPublisher will gather events coming into the webhook and publish them
	WebhookPublisher message.Publisher
	// WebhookSubscriber will subscribe to the webhook topic and handle incoming events
	WebhookSubscriber message.Subscriber
	// TODO: We'll have a Final publisher that will publish to the final topic
}

// SetUpEventer creates an Eventer object which isolates the watermill setup code
// TODO: pass in logger
func SetUpEventer() (*Eventer, error) {
	l := watermill.NewStdLogger(false, false)
	router, err := message.NewRouter(message.RouterConfig{}, l)
	if err != nil {
		return nil, err
	}

	// SignalsHandler will gracefully shutdown Router when SIGTERM is received.
	// You can also close the router by just calling `r.Close()`.
	router.AddPlugin(plugin.SignalsHandler)

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

	webhpubsub := gochannel.NewGoChannel(gochannel.Config{}, l)

	return &Eventer{
		Router:            router,
		WebhookPublisher:  webhpubsub,
		WebhookSubscriber: webhpubsub,
	}, nil
}

// Close closes the router
func (e *Eventer) Close() error {
	return e.Router.Close()
}

// Run runs the router
func (e *Eventer) Run(ctx context.Context) error {
	defer e.Close()
	return e.Router.Run(ctx)
}

// Subscribe subscribes to a topic and handles incoming messages
func (e *Eventer) Subscribe(
	topic string,
	handler message.NoPublishHandlerFunc,
) {
	e.Router.AddNoPublisherHandler(
		topic,
		topic,
		e.WebhookSubscriber,
		handler,
	)
}

// SubscribeAll subscribes to all the topics in the registrar
func (e *Eventer) SubscribeAll(r *Registrar) {
	r.Walk(func(topic string, handler message.NoPublishHandlerFunc) {
		e.Subscribe(topic, handler)
	})
}
