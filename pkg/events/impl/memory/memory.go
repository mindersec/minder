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

// Package memory provides a local in-memory transport for events.  Note
// that event persistence is entirely within the local process: if things
// crash or shut down, events could be lost.  This is mainly suitable for
// testing and development.
package memory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/stacklok/mediator/pkg/events"
)

type destinations []chan cloudevents.Event

// Broker implements an in-memory event broker which also manages
// the execution of functions supplied to it via Subscribe.
type Broker struct {
	subscriptions map[string]destinations
	subLock       sync.Mutex
	// Track the number of goroutines that need to shut down
	shutdown sync.WaitGroup
}

// New creates a new broker, with default channel depth 100.
func New() *Broker {
	return &Broker{
		subscriptions: map[string]destinations{
			"": make(destinations, 0),
		},
	}
}

var _ events.Publisher = (*Broker)(nil)
var _ events.Consumer = (*Broker)(nil)

// Subscribe implements events.Subscriber
func (b *Broker) Subscribe(_ context.Context, filter events.Filter, fn events.ConsumerFunction) error {
	// Set a reasonably-large buffer for in-memory delivery
	sink := make(chan cloudevents.Event, 100)

	b.shutdown.Add(1)

	// Subscribe starts a single background thread for in-memory.
	// Other implementations may do something fancier.
	// Note that the range call will loop until the channel is closed.
	go func() {
		for event := range sink {
			// TODO: set up context based on event content
			err := fn(context.Background(), event)
			// TODO: extract the logger from internal/logger/logging_interceptor.go and
			// inject it to both that and New here.
			if err != nil {
				// TODO: should we attempt retries?
				fmt.Printf("Error processing event %q: %s", event.ID(), err)
			}
		}
		b.shutdown.Done()
	}()

	b.subLock.Lock()
	defer b.subLock.Unlock()
	for _, eventType := range filter.EventTypes {
		b.subscriptions[eventType] = append(b.subscriptions[eventType], sink)
	}
	if len(filter.EventTypes) == 0 {
		b.subscriptions[""] = append(b.subscriptions[""], sink)
	}

	return nil
}

// Enqueue implements events.Publisher
func (b *Broker) Enqueue(_ context.Context, event cloudevents.Event) error {
	b.subLock.Lock()
	defer b.subLock.Unlock()

	if strings.Contains(event.ID(), "-must-fail") {
		return errors.New("forced failure")
	}

	// Empty string is a wildcard for all events
	for _, dest := range b.subscriptions[""] {
		dest <- event.Clone()
	}

	for _, dest := range b.subscriptions[event.Type()] {
		dest <- event.Clone()
	}

	return nil
}

// Close stops all processing on the Broker, and waits for the processing threads
// to exit. It is not valid to use the Broker after Close() is called.
func (b *Broker) Close() {
	b.subLock.Lock()
	defer b.subLock.Unlock()

	for _, dests := range b.subscriptions {
		for _, dest := range dests {
			close(dest)
		}
	}
	b.shutdown.Wait()
}
