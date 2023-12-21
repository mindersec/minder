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

package events_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/stacklok/minder/internal/config"
	"github.com/stacklok/minder/internal/events"
)

type fakeConsumer struct {
	topics            []string
	makeHandler       func(string, chan eventPair) events.Handler
	shouldFailHandler bool
	// Filled in by test later
	out chan eventPair
}

type eventPair struct {
	topic string
	msg   *message.Message
}

func driverConfig() *config.EventConfig {
	return &config.EventConfig{
		Driver:    "go-channel",
		GoChannel: config.GoChannelEventConfig{},
	}
}

func (f *fakeConsumer) Register(r events.Registrar) {
	for _, t := range f.topics {
		r.Register(t, f.makeHandler(t, f.out))
	}
}

func fakeHandler(id string, out chan eventPair) events.Handler {
	return func(msg *message.Message) error {
		ctx := msg.Context()
		select {
		case out <- eventPair{id, msg.Copy()}:
		case <-ctx.Done():
		}
		return nil
	}
}

func countFailuresHandler(counter *int) events.Handler {
	return func(_ *message.Message) error {
		*counter++
		return errors.New("handler always fails")
	}
}

func TestEventer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		publish    []eventPair
		want       map[string][]message.Message
		consumers  []fakeConsumer
		wantsCalls int
	}{
		{
			name:    "single topic",
			publish: []eventPair{{"a", &message.Message{}}},
			want:    map[string][]message.Message{"a": {}},
			consumers: []fakeConsumer{
				{topics: []string{"a"}},
			},
		},
		{
			name:    "two subscribers",
			publish: []eventPair{{"a", &message.Message{}}, {"b", &message.Message{}}, {"a", &message.Message{}}},
			want: map[string][]message.Message{
				"a": {{}, {}},
				"b": {{}},
			},
			consumers: []fakeConsumer{
				{topics: []string{"a"}},
				{topics: []string{"b"}},
			},
		},
		{
			name:    "two subscribers to topic",
			publish: []eventPair{{"a", &message.Message{}}, {"b", &message.Message{}}},
			want: map[string][]message.Message{
				"a":     {{}},
				"b":     {{}},
				"other": {{}},
			},
			consumers: []fakeConsumer{
				{topics: []string{"a", "b"}},
				{
					topics: []string{"a"},
					// This looks silly, but we need to generate a unique name for
					// the second handler on topic "a".  In real usage, each Consumer
					// will register a different function.
					makeHandler: func(_ string, out chan eventPair) events.Handler {
						return func(msg *message.Message) error {
							out <- eventPair{"other", msg.Copy()}
							return nil
						}
					},
				},
			},
		},
		{
			name:    "handler fails, message goes to DLQ",
			publish: []eventPair{{"test_dlq", &message.Message{}}},
			want: map[string][]message.Message{
				events.DeadLetterQueueTopic: {{}},
			},
			consumers: []fakeConsumer{
				{
					topics:            []string{"test_dlq"},
					shouldFailHandler: true,
				},
				{
					topics:      []string{events.DeadLetterQueueTopic},
					makeHandler: fakeHandler,
				},
			},
			wantsCalls: 4,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out := make(chan eventPair, len(tt.want))
			defer close(out)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			eventer, err := events.Setup(ctx, driverConfig())
			if err != nil {
				t.Errorf("Setup() error = %v", err)
				return
			}

			failureCounters := make([]int, len(tt.consumers))
			for i, c := range tt.consumers {
				localConsumer := c
				localIdx := i
				setupConsumer(&localConsumer, out, failureCounters, localIdx, *eventer)
			}

			go eventer.Run(ctx)
			defer eventer.Close()
			<-eventer.Running()

			for _, pair := range tt.publish {
				if err := eventer.Publish(pair.topic, pair.msg); err != nil {
					t.Errorf("Publish(%q, ...) error = %v", pair.topic, err)
					return
				}
				t.Logf("published event on %q", pair.topic)
			}
			received := make(map[string][]*message.Message, len(tt.want))
			expected := 0
			for _, msgs := range tt.want {
				expected += len(msgs)
			}
			t.Logf("Expected %d events", expected)
			for i := 0; i < expected; i++ {
				t.Logf("awaiting event %d", i)
				got := <-out
				received[got.topic] = append(received[got.topic], got.msg.Copy())
			}

			if err := eventer.Close(); err != nil {
				t.Errorf("Close() error = %v", err)
			}

			t.Log("Explicitly cancel context")
			cancel()

			for topic, msgs := range tt.want {
				if len(msgs) != len(received[topic]) {
					t.Errorf("wanted %d messages for topic %q, got %d", len(msgs), topic, len(received[topic]))
				}
			}

			for i, c := range tt.consumers {
				if c.shouldFailHandler && failureCounters[i] != tt.wantsCalls {
					t.Errorf("expected %d calls to failure handler, got %d", tt.wantsCalls, failureCounters[i])
				}
			}
		})
	}
}

func setupConsumer(c *fakeConsumer, out chan eventPair, failureCounters []int, i int, eventer events.Eventer) {
	c.out = out
	if c.makeHandler == nil {
		if c.shouldFailHandler {
			c.makeHandler = makeFailingHandler(&failureCounters[i])
		} else {
			c.makeHandler = fakeHandler
		}
	}
	eventer.ConsumeEvents(c)
}

func makeFailingHandler(counter *int) func(_ string, out chan eventPair) events.Handler {
	return func(_ string, out chan eventPair) events.Handler {
		return countFailuresHandler(counter)
	}
}
