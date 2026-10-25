// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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
	natsserver "github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/require"

	"github.com/mindersec/minder/internal/events/common"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
)

func TestNatsChannel(t *testing.T) {
	t.Parallel()
	server := natstest.RunRandClientPortServer()
	// Without a StoreDir, JetStream persists to $TMPDIR/nats, so streams and
	// unacked messages would leak between test runs.
	if err := server.EnableJetStream(&natsserver.JetStreamConfig{StoreDir: t.TempDir()}); err != nil {
		t.Fatalf("failed to enable JetStream: %v", err)
	}
	defer server.Shutdown()
	cfg := serverconfig.EventConfig{
		Nats: serverconfig.NatsConfig{
			URL:    server.ClientURL(),
			Prefix: "test",
		},
	}
	ctx := context.Background()

	// N.B. This list is in alphabetical order
	m1 := message.NewMessage("123", []byte(`{"msg":"hello"}`))
	m1.Metadata.Set("foo", "bar")
	m2 := message.NewMessage("456", []byte(`{"msg":"hola"}`))
	m3 := message.NewMessage("789", []byte(`{"msg":"konnichiwa"}`))
	m4 := message.NewMessage("456", []byte(`{"msg":"ni hao"}`))

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
	// Publish waits for the JetStream ack, but delivery to subscribers is
	// asynchronous. Closing sub1 doesn't wait for its subscription to drain,
	// so a message pushed to sub1 but not yet acked would only be redelivered
	// after the ack wait (30s). Wait until every message has been acked.
	waitForAllAcked(ctx, t, server.ClientURL())
	// Don't let sub1 see the last two messages, even though it's published by pub1.
	if err := sub1.Close(); err != nil {
		t.Fatalf("failed to close sub1: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if err := pub2.Publish("test", m3); err != nil {
		t.Fatalf("failed to publish message: %v", err)
	}
	if err := pub1.Publish("test", m4); err != nil {
		t.Fatalf("failed to publish message: %v", err)
	}

	// By reading the above 4 messages off of two subscribers, we aim to verify:
	// 1. Every message is delivered. Duplicates (redeliveries) are tolerated and
	//    removed below, so this does not verify exactly-once delivery.
	// 2. Messages are delivered across publisher/subscriber pairs.
	// 3. Message payloads match what they were sent with.
	//
	// Note that message delivery order is not important (and may not be deterministic /
	// meaningful in a multi-process world).
	results := make([]*message.Message, 0, 7)
loop:
	for i := 0; len(results) < 4; i++ {
		select {
		case m := <-out1:
			results = append(results, m)
			t.Logf("Got %s from out1", m.Payload)
		case m := <-out2:
			results = append(results, m)
			t.Logf("Got %s from out2", m.Payload)
		case <-time.After(5 * time.Second):
			t.Logf("timeout waiting for message %d", i)
			break loop
		}

		slices.SortFunc(results, func(a, b *message.Message) int {
			return bytes.Compare(a.Payload, b.Payload)
		})
		// We sometimes get message duplicates.  Retransmissions are okay; deduplicate them.
		results = slices.CompactFunc(results, func(a, b *message.Message) bool {
			return bytes.Equal(a.Payload, b.Payload)
		})
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 messages, got %d: %+v", len(results), results)
	}

	expectMessageEqual(t, m1, results[0])
	expectMessageEqual(t, m2, results[1])
	expectMessageEqual(t, m3, results[2])
	expectMessageEqual(t, m4, results[3])
}

// Closing one subscriber must not delete the durable consumer that the other
// subscribers in the queue group are bound to.
func TestNatsChannelSubscriberCloseKeepsConsumer(t *testing.T) {
	t.Parallel()
	server := natstest.RunRandClientPortServer()
	require.NoError(t, server.EnableJetStream(&natsserver.JetStreamConfig{StoreDir: t.TempDir()}))
	defer server.Shutdown()
	cfg := serverconfig.EventConfig{
		Nats: serverconfig.NatsConfig{
			URL:    server.ClientURL(),
			Prefix: "test",
		},
	}
	ctx := context.Background()

	conn, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer conn.Close()
	js, err := jetstream.New(conn)
	require.NoError(t, err)
	consumerExists := func() bool {
		_, err := js.PushConsumer(ctx, "test", "test-test")
		return err == nil
	}

	// Cancelling the subscription context drains the subscription while the
	// connection stays open, as happens when the server shuts down.
	ctx1, cancel1 := context.WithCancel(ctx)
	defer cancel1()
	_, sub1, closer1, _, err := buildDriverPair(ctx1, cfg)
	require.NoError(t, err)
	defer closer1()
	require.Eventually(t, consumerExists, 5*time.Second, 5*time.Millisecond, "consumer was not created")

	pub2, _, closer2, out2, err := buildDriverPair(ctx, cfg)
	require.NoError(t, err)
	defer closer2()
	// Subscribe returns before the subscription is made, so wait until both
	// subscribers have joined the queue group on the consumer's deliver subject.
	info, err := js.PushConsumer(ctx, "test", "test-test")
	require.NoError(t, err)
	deliver := info.CachedInfo().Config.DeliverSubject
	require.Eventually(t, func() bool {
		connz, err := server.Connz(&natsserver.ConnzOptions{Subscriptions: true})
		if err != nil {
			return false
		}
		members := 0
		for _, c := range connz.Conns {
			if slices.Contains(c.Subs, deliver) {
				members++
			}
		}
		return members == 2
	}, 5*time.Second, 5*time.Millisecond, "subscribers did not join the queue group")

	cancel1()
	require.Never(t, func() bool { return !consumerExists() }, 500*time.Millisecond, 10*time.Millisecond,
		"draining the first subscription deleted the consumer")
	require.NoError(t, sub1.Close())
	require.True(t, consumerExists(), "closing the first subscriber deleted the consumer")

	want := message.NewMessage("123", []byte(`{"msg":"hello"}`))
	require.NoError(t, pub2.Publish("test", want))
	select {
	case got := <-out2:
		expectMessageEqual(t, want, got)
	case <-time.After(5 * time.Second):
		t.Fatal("second subscriber did not receive a message after the first one closed")
	}
}

// Publishing to a topic before subscribing to it from the same adapter must not
// leave the subscriber on the publish-side protocol, which doesn't bind to the
// explicitly created consumer.
func TestNatsChannelPublishBeforeSubscribe(t *testing.T) {
	t.Parallel()
	server := natstest.RunRandClientPortServer()
	require.NoError(t, server.EnableJetStream(&natsserver.JetStreamConfig{StoreDir: t.TempDir()}))
	defer server.Shutdown()
	cfg := serverconfig.EventConfig{
		Nats: serverconfig.NatsConfig{
			URL:    server.ClientURL(),
			Prefix: "test",
		},
	}
	ctx := context.Background()

	conn, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer conn.Close()
	js, err := jetstream.New(conn)
	require.NoError(t, err)

	pub, sub, closer, err := BuildNatsChannelDriver(&cfg)
	require.NoError(t, err)
	defer closer()
	defer func() { _ = pub.Close() }()

	first := message.NewMessage("123", []byte(`{"msg":"hello"}`))
	require.NoError(t, pub.Publish("test", first))
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	out, err := sub.Subscribe(subCtx, "test")
	require.NoError(t, err)

	// Subscribe returns before the subscription is made, so wait until the
	// subscriber has joined the queue group on the consumer's deliver subject.
	consumer, err := js.PushConsumer(ctx, "test", "test-test")
	require.NoError(t, err, "Subscribe did not create the consumer")
	deliver := consumer.CachedInfo().Config.DeliverSubject
	require.Eventually(t, func() bool {
		connz, err := server.Connz(&natsserver.ConnzOptions{Subscriptions: true})
		if err != nil {
			return false
		}
		for _, c := range connz.Conns {
			if slices.Contains(c.Subs, deliver) {
				return true
			}
		}
		return false
	}, 5*time.Second, 5*time.Millisecond, "subscriber did not bind to the consumer")

	second := message.NewMessage("456", []byte(`{"msg":"hola"}`))
	require.NoError(t, pub.Publish("test", second))
	got := make([]*message.Message, 0, 2)
	for len(got) < 2 {
		select {
		case m := <-out:
			if !slices.ContainsFunc(got, func(g *message.Message) bool { return bytes.Equal(g.Payload, m.Payload) }) {
				got = append(got, m)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("expected 2 messages, got %d", len(got))
		}
	}
	slices.SortFunc(got, func(a, b *message.Message) int { return bytes.Compare(a.Payload, b.Payload) })
	expectMessageEqual(t, first, got[0])
	expectMessageEqual(t, second, got[1])

	// The only consumer is the explicitly created one.
	stream, err := js.Stream(ctx, "test")
	require.NoError(t, err)
	names := stream.ConsumerNames(ctx)
	var consumers []string
	for name := range names.Name() {
		consumers = append(consumers, name)
	}
	require.NoError(t, names.Err())
	require.Equal(t, []string{"test-test"}, consumers)

	cancel()
	require.NoError(t, sub.Close())
	_, err = js.PushConsumer(ctx, "test", "test-test")
	require.NoError(t, err, "closing the subscriber deleted the consumer")
}

// A consumer created by another process between the lookup and the create, with
// a different config (old nats.go used a random deliver subject), is reused.
func TestCreateConsumerReusesExisting(t *testing.T) {
	t.Parallel()
	server := natstest.RunRandClientPortServer()
	require.NoError(t, server.EnableJetStream(&natsserver.JetStreamConfig{StoreDir: t.TempDir()}))
	defer server.Shutdown()
	ctx := context.Background()

	conn, err := nats.Connect(server.ClientURL())
	require.NoError(t, err)
	defer conn.Close()
	js, err := jetstream.New(conn)
	require.NoError(t, err)
	_, err = js.CreateStream(ctx, jetstream.StreamConfig{Name: "test", Subjects: []string{"test.>"}})
	require.NoError(t, err)

	inbox := nats.NewInbox()
	_, err = js.CreatePushConsumer(ctx, "test", jetstream.ConsumerConfig{
		Durable:        "test-test",
		DeliverSubject: inbox,
		DeliverGroup:   "test-test",
		FilterSubject:  "test.test",
		AckPolicy:      jetstream.AckExplicitPolicy,
	})
	require.NoError(t, err)

	require.NoError(t, createConsumer(ctx, js, "test", "test.test", "test-test"))
	consumer, err := js.PushConsumer(ctx, "test", "test-test")
	require.NoError(t, err)
	require.Equal(t, inbox, consumer.CachedInfo().Config.DeliverSubject)
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
	// The driver's own closer is a no-op, so also close the adapter. Otherwise its
	// NATS connections keep reconnecting after the server shuts down, and can join
	// a later test run's server if it reuses the same port.
	closeAll := func() {
		closer()
		_ = pub.Close()
	}
	return pub, sub, closeAll, out, nil
}

// waitForAllAcked waits until the consumer has acked every message in the
// stream. The consumer's NumPending isn't enough, because the server counts a
// new message asynchronously, after Publish has returned.
func waitForAllAcked(ctx context.Context, t *testing.T, url string) {
	t.Helper()
	conn, err := nats.Connect(url)
	require.NoError(t, err)
	defer conn.Close()
	js, err := jetstream.New(conn)
	require.NoError(t, err)
	stream, err := js.Stream(ctx, "test")
	require.NoError(t, err)
	consumer, err := js.PushConsumer(ctx, "test", "test-test")
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		streamInfo, err := stream.Info(ctx)
		if err != nil {
			return false
		}
		consumerInfo, err := consumer.Info(ctx)
		if err != nil {
			return false
		}
		return consumerInfo.AckFloor.Stream >= streamInfo.State.LastSeq
	}, 10*time.Second, 5*time.Millisecond, "not every message was acked")
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
