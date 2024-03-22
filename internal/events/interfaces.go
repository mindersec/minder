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

package events

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
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

// Publisher is an interface implemented by components which wish to publish events.
type Publisher interface {
	// Publish implements message.Publisher
	Publish(topic string, messages ...*message.Message) error
}

// Service is an interface that allows the eventer to orchestrate the
// consumption and publication of events, as well as start and stop the
// event router.
type Service interface {
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

// Interface is a combination of the Publisher, Registrar, and Service interfaces.
// This is handy when spawning the eventer in a single function for easy setup.
type Interface interface {
	Publisher
	Registrar
	Service
}
