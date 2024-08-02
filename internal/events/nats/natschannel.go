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
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
	cejsm "github.com/cloudevents/sdk-go/protocol/nats_jetstream/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/events/common"
)

// BuildNatsChannelDriver creates a new event driver using
// CloudEvents with the NATS-JetStream transport
func BuildNatsChannelDriver(cfg *serverconfig.EventConfig) (message.Publisher, message.Subscriber, common.DriverCloser, error) {
	adapter := &cloudEventsNatsAdapter{cfg: &cfg.Nats}
	return adapter, adapter, func() {}, nil
}

// CloudEventsNatsPublisher actually consumes a _set_ of NATS topics,
// because CloudEvents-Jetstream has a separate Consumer for each topic
type cloudEventsNatsAdapter struct {
	cfg    *serverconfig.NatsConfig
	lock   sync.Mutex
	// Keep a cache of the topics we subscribe/publish to
	topics map[string]topicState
}

type topicState struct {
	ceProtocol cejsm.Protocol
	ceClient   cloudevents.Client
}

var _ message.Subscriber = (*cloudEventsNatsAdapter)(nil)

var _ message.Publisher = (*cloudEventsNatsAdapter)(nil)

// Close implements message.Subscriber and message.Publisher.
func (c *cloudEventsNatsAdapter) Close() error {
	zerolog.Ctx(context.Background()).Info().Msg("Closing NATS event driver")
	c.lock.Lock()
	defer c.lock.Unlock()
	// We want to try to close all the consumers, but some may fail
	var gotErr error
	for topic, state := range c.topics {
		err := state.ceProtocol.Close(context.Background())
		if err != nil {
			gotErr = err
		} else {
			delete(c.topics, topic)
		}
	}
	return gotErr
}

// Ensure that we have a consumer for this topic
func (c *cloudEventsNatsAdapter) ensureTopic(ctx context.Context, topic string, queue string) (*topicState, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.topics == nil {
		c.topics = make(map[string]topicState)
	}

	state, ok := c.topics[topic]
	if ok {
		return &state, nil
	}
	opts := []nats.Option{
		nats.Name("minder"),
		// TODO: set TLS config
		// TODO: set UserJWT
	}
	jetstreamOpts := []nats.JSOpt{}
	subOpts := []nats.SubOpt{}
	// Pre-create the Stream to work around the SDK creating it as "stream.*" rather then "stream.>"
	// We can remove this after https://github.com/cloudevents/sdk-go/pull/1084 is merged and released
	if err := c.ensureStream(ctx); err != nil {
		return nil, err
	}

	consumer, err := cejsm.NewProtocol(
		c.cfg.URL, c.cfg.Prefix, topic, topic,
		opts, jetstreamOpts, subOpts,
		cejsm.WithConsumerOptions(cejsm.WithQueueSubscriber(queue)))
	if err != nil {
		return nil, err
	}

	ceSub, err := cloudevents.NewClient(consumer)
	if err != nil {
		_ = consumer.Close(ctx)
		return nil, err
	}
	state = topicState{
		ceProtocol: *consumer,
		ceClient:   ceSub,
	}
	c.topics[topic] = state
	return &state, nil
}

func (c *cloudEventsNatsAdapter) ensureStream(ctx context.Context) error {
	conn, err := nats.Connect(c.cfg.URL)
	if err != nil {
		return err
	}
	defer conn.Close()
	js, err := conn.JetStream()
	if err != nil {
		return err
	}
	si, err := js.StreamInfo(c.cfg.Prefix)
	if si == nil || err != nil && err.Error() == "stream not found" {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     c.cfg.Prefix,
			Subjects: []string{c.cfg.Prefix + ".>"},
		})
		return err
	}
	return nil
}

// Subscribe implements message.Subscriber.
func (c *cloudEventsNatsAdapter) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	subject := fmt.Sprintf("%s.%s", c.cfg.Prefix, topic)
	// consumer names cannot contain "."
	queueConsumer := strings.ReplaceAll(subject, ".", "-")

	// TODO: should we have separate maps for producer & consumer?  I've lazily combined them here.
	state, err := c.ensureTopic(ctx, subject, queueConsumer)
	if err != nil {
		return nil, err
	}

	out := make(chan *message.Message)
	err = state.ceClient.StartReceiver(ctx, convertCloudEventToMessage(out))
	if err != nil {
		err = fmt.Errorf("Error subscribing to topic %q: %w", subject, err)
	}
	return out, err
}

func convertCloudEventToMessage(outChan chan *message.Message) func(ctx context.Context, event cloudevents.Event) error {
	return func(ctx context.Context, event cloudevents.Event) error {
		msg := message.NewMessage(event.ID(), event.Data())
		msg.SetContext(ctx)
		// Add some extra message metadata from the CloudEvent
		msg.Metadata.Set("ce-id", event.ID())
		msg.Metadata.Set("ce-source", event.Source())
		msg.Metadata.Set("ce-type", event.Type())
		msg.Metadata.Set("ce-subject", event.Subject())
		msg.Metadata.Set("ce-time", event.Time().String())
		msg.Metadata.Set("ce-datacontenttype", event.DataContentType())
		msg.Metadata.Set("ce-schemaurl", event.DataSchema())

		for k, v := range event.Extensions() {
			// Strip "minder" prefix from metadata keys if present
			// The prefix avoids collision on keys like "type"
			k = strings.TrimPrefix(k, "minder")
			// Undo the transformation from 228 in sendEvent
			k = strings.ReplaceAll(k, "0", "_")
			msg.Metadata.Set(k, fmt.Sprintf("%s", v))
		}

		outChan <- msg
		return nil
	}
}

// Publish implements message.Publisher.
func (c *cloudEventsNatsAdapter) Publish(topic string, messages ...*message.Message) error {
	ctx := context.Background()
	subject := fmt.Sprintf("%s.%s", c.cfg.Prefix, topic)

	state, err := c.ensureTopic(ctx, subject, "sender") // subject)
	if err != nil {
		return fmt.Errorf("Error creating topic %q: %w", subject, err)
	}

	for _, msg := range messages {
		err := sendEvent(ctx, subject, state.ceClient, msg)
		if err != nil {
			return fmt.Errorf("Error sending event to %q: %w", subject, err)
		}
	}
	return nil
}

func sendEvent(
	ctx context.Context, eventType string, ceClient cloudevents.Client, msg *message.Message) error {
	event := cloudevents.NewEvent()
	event.SetID(msg.UUID)
	event.SetType(eventType)
	event.SetSource("minder") // The system which generated the event.  The Minder URL would be nice here.
	event.SetSubject("TODO")  // This *should* represent the entity, but we don't have a standard field for it yet.

	// All our current payloads are JSON
	err := event.SetData("application/json", msg.Payload)
	if err != nil {
		return err
	}
	for k, v := range msg.Metadata {
		// CloudEvents does not allow "_" or "-" in attribute keys, only A-Z, a-z, 0-9.
		ceKey := strings.ReplaceAll(k, "_", "0")
		event.SetExtension("minder"+ceKey, v)
	}

	return ceClient.Send(ctx, event)
}
