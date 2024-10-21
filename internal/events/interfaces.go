// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

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
