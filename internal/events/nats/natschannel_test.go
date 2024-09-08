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

// Package nats provides a nants+cloudevents implementation of the eventer interface
package nats

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	natsserver "github.com/nats-io/nats-server/v2/test"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/events/common"
)

func TestNatsChannel(t *testing.T) {
	t.Parallel()
	server := natsserver.RunRandClientPortServer()
	if err := server.EnableJetStream(nil); err != nil {
		t.Fatalf("failed to enable JetStream: %v", err)
	}
	defer server.Shutdown()
	cfg := serverconfig.EventConfig{
		Nats: serverconfig.NatsConfig{
			URL:    server.ClientURL(),
			Prefix: "test",
			Queue:  "minder",
		},
	}
	ctx := context.Background()

	// N.B. This list is in alphabetical order
	m1 := message.NewMessage("123", []byte(`{"msg":"hello"}`))
	m1.Metadata.Set("foo", "bar")
	m2 := message.NewMessage("456", []byte(`{"msg":"hola"}`))
	m3 := message.NewMessage("789", []byte(`{"msg":"konnichiwa"}`))

	pub1, sub1, closer1, out1, err := buildDriverPair(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to build nats channel driver: %v", err)
	}
	defer closer1()

	// Publish one message before the second driver is created
	if err := pub1.Publish("test", m1); err != nil {
		t.Fatalf("failed to publish message: %v", err)
	}

	pub2, _, closer2, out2, err := buildDriverPair(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to build nats channel driver: %v", err)
	}
	defer closer2()

	if err := pub2.Publish("test", m2); err != nil {
		t.Fatalf("failed to publish message: %v", err)
	}
	// Don't let sub1 see the last message, even though it's published by pub1.
	if err := sub1.Close(); err != nil {
		t.Fatalf("failed to close sub1: %v", err)
	}
	if err := pub1.Publish("test", m3); err != nil {
		t.Fatalf("failed to publish message: %v", err)
	}

	// By reading the above 3 messages off of two subscribers, we aim to verify:
	// 1. Messages are delivered once -- if both subscribers get the message, we should end up
	//    with duplicate messages in the array, which will fail the test.
	// 2. Messages are delivered across publisher/subscriber pairs.
	// 3. Message payloads match what they were sent with.
	//
	// Note that message delivery order is not important (and may not be deterministic /
	// meaningful in a multi-process world).
	results := make([]*message.Message, 0, 7)
	for i := 0; i < 7; i++ {
		select {
		case m := <-out1:
			results = append(results, m)
			t.Logf("Got %s from out1", m.Payload)
		case m := <-out2:
			results = append(results, m)
			t.Logf("Got %s from out2", m.Payload)
		case <-time.After(30 * time.Second):
			t.Fatalf("timeout waiting for message %d", i)
		}
	}
	slices.SortFunc(results, func(a, b *message.Message) int {
		return bytes.Compare(a.Payload, b.Payload)
	})
	// We sometimes get message duplicates.  Retransmissions are okay; deduplicate them.
	results = slices.CompactFunc(results, func(a, b *message.Message) bool {
		return bytes.Equal(a.Payload, b.Payload)
	})

	if len(results) != 3 {
		t.Fatalf("expected 3 messages, got %d: %+v", len(results), results)
	}

	expectMessageEqual(t, m1, results[0])
	expectMessageEqual(t, m2, results[1])
	expectMessageEqual(t, m3, results[2])
}

func buildDriverPair(ctx context.Context, cfg serverconfig.EventConfig) (message.Publisher, message.Subscriber, common.DriverCloser, <-chan *message.Message, error) {
	pub, sub, closer, err := BuildNatsChannelDriver(&cfg)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to build nats channel driver: %v", err)
	}
	out, err := sub.Subscribe(ctx, "test")
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to subscribe: %v", err)
	}
	return pub, sub, closer, out, nil
}

func expectMessageEqual(t *testing.T, want *message.Message, got *message.Message) {
	t.Helper()
	if !bytes.Equal(want.Payload, got.Payload) {
		t.Errorf("expected %v, got %v", string(want.Payload), string(got.Payload))
	}
	// got will have a bunch of additional CloudEvents metadata
	if want.Metadata["foo"] != got.Metadata["foo"] {
		t.Errorf("expected %v, got %v", want.Metadata, got.Metadata)
	}
	if got.Metadata["ce-time"] == "" {
		t.Errorf("expected ce-time to be set, got %v", got.Metadata)
	}
}
