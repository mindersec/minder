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
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stacklok/mediator/internal/events"
)

type fakeConsumer struct {
	topics      []string
	makeHandler func(string, chan eventPair) events.Handler
	// Filled in by test later
	out chan eventPair
}

type eventPair struct {
	topic string
	msg   *message.Message
}

func (f *fakeConsumer) Register(r events.Registrar) {
	for _, t := range f.topics {
		r.Register(t, f.makeHandler(t, f.out))
	}
}

func fakeHandler(id string, out chan eventPair) events.Handler {
	return func(msg *message.Message) error {
		out <- eventPair{id, msg.Copy()}
		return nil
	}
}
func TestEventer(t *testing.T) {

	tests := []struct {
		name      string
		publish   []eventPair
		want      map[string][]message.Message
		consumers []fakeConsumer
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eventer, err := events.Setup()
			if err != nil {
				t.Errorf("Setup() error = %v", err)
				return
			}

			out := make(chan eventPair, len(tt.want))
			defer close(out)
			for _, c := range tt.consumers {
				c.out = out
				if c.makeHandler == nil {
					c.makeHandler = fakeHandler
				}
				local := c // Avoid aliasing
				eventer.ConsumeEvents(&local)
			}

			go eventer.Run(context.Background())
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
			for topic, msgs := range tt.want {
				if len(msgs) != len(received[topic]) {
					t.Errorf("wanted %d messages for topic %q, got %d", len(msgs), topic, len(received[topic]))
				}
			}
		})
	}
}
