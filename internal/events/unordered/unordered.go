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

// Package unordered providers a handler wrapper that will handle events in an unordered fashion.
// This means that events will be acked immediately, and the handler will not wait for the event to be processed.
// If a retry is needed, the event will be requeued and processed again.
package unordered

import (
	"context"
	"strconv"
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/rs/zerolog"

	"github.com/stacklok/minder/internal/events"
)

const (
	// MessageRepublishedMetadataKey is the key used to store the republished metadata
	MessageRepublishedMetadataKey = "unordered_processor_republished"
	// MetadataRepublishRetriesKey is the key used to store the number of times the message has been republished
	MetadataRepublishRetriesKey = "unordered_processor_republish_retries"

	// DefaultMaxRetries is the default number of times a message will be retried
	DefaultMaxRetries = 3
)

// Retrier is a handler wrapper that will handle event retries in an unordered fashion.
type Retrier struct {
	pub events.Publisher
	wg  sync.WaitGroup
}

// New creates a new UnorderedProcessor
func New(pub events.Publisher) *Retrier {
	return &Retrier{pub: pub}
}

// Wrap wraps the handler with the unordered processor
func (up *Retrier) Wrap(topic string, h events.Handler) events.Handler {
	return func(msg *message.Message) error {
		ctx := msg.Context()
		newMsg := msg.Copy()

		up.wg.Add(1)
		go func() {
			defer up.wg.Done()

			err := h(newMsg)
			if err == nil {
				return
			}

			newMsg = setMetadata(newMsg)
			republishIfRequired(topic, newMsg, up.pub,
				buildLogger(ctx, topic, msg.UUID, err))
		}()
		return nil
	}
}

// Wait waits for all the messages to be processed
func (up *Retrier) Wait() {
	up.wg.Wait()
}

func setMetadata(msg *message.Message) *message.Message {
	msg.Metadata.Set(MessageRepublishedMetadataKey, "true")
	if retries := msg.Metadata.Get(MetadataRepublishRetriesKey); retries != "" {
		r, err := strconv.Atoi(retries)
		if err != nil {
			r = 0
		}

		msg.Metadata.Set(MetadataRepublishRetriesKey, strconv.Itoa(r+1))
	} else {
		msg.Metadata.Set(MetadataRepublishRetriesKey, "1")
	}

	return msg
}

func republishIfRequired(topic string, msg *message.Message, pub events.Publisher, l zerolog.Logger) {
	if !retriable(msg) {
		l.Error().Msg("message not retriable. dropping message")
		return
	}

	republish(topic, msg, pub, l)
}

func republish(topic string, msg *message.Message, pub events.Publisher, l zerolog.Logger) {
	// republish the message
	if err := pub.Publish(topic, msg); err != nil {
		l.Error().Msg("failed to republish message")
	}
}

func retriable(msg *message.Message) bool {
	r, err := strconv.Atoi(msg.Metadata.Get(MetadataRepublishRetriesKey))
	if err != nil {
		return false
	}

	return r <= DefaultMaxRetries
}

func buildLogger(ctx context.Context, topic string, msgID string, upstreamErr error) zerolog.Logger {
	return zerolog.Ctx(ctx).With().
		Str("component", "unordered-processor").
		Str("topic", topic).
		Str("message-id", msgID).
		Err(upstreamErr).
		Logger()
}
