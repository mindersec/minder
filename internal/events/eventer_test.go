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
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	serverconfig "github.com/stacklok/minder/internal/config/server"
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

func driverConfig() *serverconfig.EventConfig {
	return &serverconfig.EventConfig{
		Driver:    "go-channel",
		GoChannel: serverconfig.GoChannelEventConfig{},
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
			publish: []eventPair{{"a", &message.Message{Metadata: map[string]string{}}}},
			want:    map[string][]message.Message{"a": {}},
			consumers: []fakeConsumer{
				{topics: []string{"a"}},
			},
		},
		{
			name: "two subscribers",
			publish: []eventPair{
				{"a", &message.Message{Metadata: map[string]string{"one": "1"}}},
				{"b", &message.Message{Metadata: map[string]string{"two": "2"}}},
				{"a", &message.Message{Metadata: map[string]string{"three": "3"}}},
			},
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
			name: "two subscribers to topic",
			publish: []eventPair{
				{"a", &message.Message{Metadata: map[string]string{"one": "1"}}},
				{"b", &message.Message{Metadata: map[string]string{"two": "2"}}},
			},
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
			publish: []eventPair{{"test_dlq", &message.Message{Metadata: map[string]string{}}}},
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

			eventer, metricReader, err := setupEventerWithMetricReader(ctx)
			if err != nil {
				t.Errorf("Setup() error = %v", err)
				return
			}

			failureCounters := make([]int, len(tt.consumers))
			for i, c := range tt.consumers {
				localConsumer := c
				localIdx := i
				setupConsumer(&localConsumer, out, &failureCounters[localIdx])
				eventer.ConsumeEvents(&localConsumer)
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

			// Metrics collection happens in another goroutine, so make sure it has a chance to run.
			time.Sleep(2 * time.Millisecond)

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

			// Sending messages to the DLQ also counts for processing, so add those in:
			expectedFailures := 0
			for _, c := range failureCounters {
				if c > 0 {
					expectedFailures++
				}
			}
			checkEventCounts(t, metricReader, uint64(expected), uint64(expectedFailures))
		})
	}
}

var setupMu sync.Mutex

// We currently use the global meter provider, so reset it for each test.
// Since this is global, we use a global mutex to ensure we don't enter setup
// concurrently.
func setupEventerWithMetricReader(ctx context.Context) (events.Interface, *metric.ManualReader, error) {
	setupMu.Lock()
	defer setupMu.Unlock()
	oldMeter := otel.GetMeterProvider()
	defer otel.SetMeterProvider(oldMeter)
	metricReader := metric.NewManualReader()
	otel.SetMeterProvider(metric.NewMeterProvider(metric.WithReader(metricReader)))
	eventer, err := events.Setup(ctx, driverConfig())
	return eventer, metricReader, err
}

func setupConsumer(c *fakeConsumer, out chan eventPair, failureCounter *int) {
	c.out = out
	if c.makeHandler == nil {
		if c.shouldFailHandler {
			c.makeHandler = makeFailingHandler(failureCounter)
		} else {
			c.makeHandler = fakeHandler
		}
	}
}

func makeFailingHandler(counter *int) func(_ string, _ chan eventPair) events.Handler {
	return func(_ string, _ chan eventPair) events.Handler {
		return countFailuresHandler(counter)
	}
}

func checkEventCounts(t *testing.T, reader *metric.ManualReader, expectedOk uint64, expectedFail uint64) {
	t.Helper()
	rm := metricdata.ResourceMetrics{}
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("Unable to read metrics: %v", err)
	}
	if expectedOk == 0 && expectedFail == 0 {
		return
	}

	if len(rm.ScopeMetrics) != 1 {
		t.Fatalf("Expected 1 scope metric, got %d", len(rm.ScopeMetrics))
	}
	if len(rm.ScopeMetrics[0].Metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(rm.ScopeMetrics[0].Metrics))
	}
	if rm.ScopeMetrics[0].Metrics[0].Name != "messages.processing_delay" {
		t.Errorf("Expected 'messages.processing_delay' metric, got %q", rm.ScopeMetrics[0].Metrics[0].Name)
	}
	h, ok := rm.ScopeMetrics[0].Metrics[0].Data.(metricdata.Histogram[int64])
	if !ok {
		t.Fatalf("Expected histogram data, got %T", rm.ScopeMetrics[0].Metrics[0].Data)
	}
	if len(h.DataPoints) != 1 {
		t.Fatalf("Expected only one data point, got %v", h)
	}

	if poisonVal, ok := h.DataPoints[0].Attributes.Value("poison"); !ok {
		t.Errorf("Doesn't have 'poison' attribute")
	} else {
		// In our simple test cases, if we have failures, we expect all messages to be poison.
		if poisonVal.AsBool() != (expectedFail > 0) {
			t.Errorf("Expected poison attribute to be %v, got %v", expectedFail > 0, poisonVal.AsBool())
		}
	}

	allCounts := uint64(0)
	for _, c := range h.DataPoints[0].BucketCounts {
		allCounts += c
	}
	if allCounts != expectedOk+expectedFail {
		t.Errorf("Expected %d messages, got %d", expectedOk+expectedFail, allCounts)
	}

	largestBucket := h.DataPoints[0].Bounds[len(h.DataPoints[0].Bounds)-1]
	if largestBucket < 5*60*1000 {
		t.Errorf("Expected largest bucket to be at least 5 minutes, was %f", largestBucket)
	}
}
