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

package memory_test

import (
	"context"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/go-cmp/cmp"
	"github.com/stacklok/mediator/pkg/events"
	"github.com/stacklok/mediator/pkg/events/impl/memory"
)

// TODO: matrixify
func TestSimple(t *testing.T) {
	b := memory.New()

	send := []cloudevents.Event{
		{
			Context: &cloudevents.EventContextV1{
				ID:   "okay",
				Type: "example.event",
			},
		},
	}
	expect := append([]cloudevents.Event(nil), send...)
	got := make([]cloudevents.Event, 0, len(expect))

	recv := func(_ context.Context, e cloudevents.Event) error {
		got = append(got, e)
		return nil
	}
	if err := b.Subscribe(context.Background(), events.Filter{}, recv); err != nil {
		t.Fatal(err)
	}

	for _, event := range send {
		if err := b.Enqueue(context.Background(), event); err != nil {
			t.Fatal(err)
		}
	}
	b.Close()
	for i, event := range expect {
		if len(got) <= i {
			t.Fatalf("Failed to receive event %d", i)
		}
		if diff := cmp.Diff(event, got[i]); diff != "" {
			t.Errorf("Events do not match: %s", diff)
		}
	}
}

// TODO: refactor this; it tests three things:
// 1. Filtering
// 2. Fanout to two functions
// 3. Modification of events
func TestFilter(t *testing.T) {
	b := memory.New()

	send := []cloudevents.Event{
		{
			Context: &cloudevents.EventContextV1{
				ID:   "nomatch",
				Type: "other.event",
			},
		},
		{
			Context: &cloudevents.EventContextV1{
				ID:   "okay",
				Type: "example.event",
			},
		},
	}
	expect := append([]cloudevents.Event(nil), send[1])
	got := make([]cloudevents.Event, 0, len(expect))
	all := make([]cloudevents.Event, 0, len(send))

	recv := func(_ context.Context, e cloudevents.Event) error {
		got = append(got, e)
		return nil
	}
	if err := b.Subscribe(context.Background(), events.Filter{EventTypes: []string{"example.event"}}, recv); err != nil {
		t.Fatal(err)
	}
	matchAll := func(_ context.Context, e cloudevents.Event) error {
		if err := e.SetData("text/plain", "modified"); err != nil {
			t.Fatal(err)
		}
		all = append(all, e)
		return nil
	}
	if err := b.Subscribe(context.Background(), events.Filter{}, matchAll); err != nil {
		t.Fatal(err)
	}

	for _, event := range send {
		if err := b.Enqueue(context.Background(), event); err != nil {
			t.Fatal(err)
		}
	}
	b.Close()
	for i, event := range expect {
		if len(got) <= i {
			t.Fatalf("Failed to receive event %d", i)
		}
		if diff := cmp.Diff(event, got[i]); diff != "" {
			t.Errorf("Events do not match: %s", diff)
		}
	}
	for i, event := range send {
		if len(all) <= i {
			t.Fatalf("Failed to receive event %d", i)
		}
		if err := event.SetData("text/plain", "modified"); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(event, all[i]); diff != "" {
			t.Errorf("Events do not match: %s", diff)
		}
	}
}

// TODO: test failed Enqueue
