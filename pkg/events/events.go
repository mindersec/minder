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

// Package events provides abstract implementations for event distribution.
// You will want to use exactly one of the concrete implementations under
// pkg/events/impl to fulfil this interface.
package events

import (
	"context"

	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// Publisher provides the ability to publish events.  When Enqueue returns
// with non-nil error, the event has been stored in the platform.  If error
// is set, the event may or may not have been published, but should be
// retried if possible
type Publisher interface {
	Enqueue(ctx context.Context, event cloudevents.Event) error
}

// ConsumerFunction is a function which receives CloudEvents representing
// state changes (such as various GitHub event types).
type ConsumerFunction func(context.Context, cloudevents.Event) error

// Filter specifies a desired message filter on the stream of incoming events.
// This type _should_ remain simple to ensure maximum flexibility across
// implementations.  Currently, we only support matching on one or more event
// types.  An empty filter matches all events.
type Filter struct {
	// Exact match on event types
	EventTypes []string
}

// Consumer provides the ability to consume events matching a filter.
// For each message matching filter, func will be called **one or more**
// times to process the event.  At least one func call must return a nil
// error for the event to be considered processed.
type Consumer interface {
	// assume a “filter” type which contains an exact string match on event type for now.
	// Ideally, we could encode filter like an annotation, but this is Go, so...
	Subscribe(ctx context.Context, filter Filter, fn ConsumerFunction) error
}
