// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/mindersec/minder/internal/engine/engcontext"
	"github.com/mindersec/minder/internal/events/common"
	"github.com/mindersec/minder/pkg/flags"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
)

type flaggedDriver struct {
	basePub       message.Publisher
	experimentPub message.Publisher
	baseSub       message.Subscriber
	experimentSub message.Subscriber

	flags             openfeature.IClient
	publishedMessages metric.Int64Counter
	readMessages      metric.Int64Counter
}

// Publish implements message.Publisher.  If the message is flagged by the
// alternate_message_driver experiment, it is published to the experiment
// driver, otherwise it is published to the base driver.
func (f *flaggedDriver) Publish(topic string, messages ...*message.Message) error {
	// Each message has its own context, so they _could_ be in different flag treatments
	emptyContext := engcontext.EntityContext{}
	for _, m := range messages {
		if engcontext.EntityFromContext(m.Context()) == emptyContext {
			zerolog.Ctx(m.Context()).Warn().Str("stack", string(debug.Stack())).Msg("No entity in context")
		}
		if flags.Bool(m.Context(), f.flags, flags.AlternateMessageDriver) {
			if err := f.experimentPub.Publish(topic, m); err != nil {
				return err
			}
			f.publishedMessages.Add(m.Context(), 1, metric.WithAttributes(attribute.Bool("experiment", true)))
		} else {
			if err := f.basePub.Publish(topic, m); err != nil {
				return err
			}
			f.publishedMessages.Add(m.Context(), 1, metric.WithAttributes(attribute.Bool("experiment", false)))
		}
	}
	return nil
}

// Subscribe implements message.Subscriber.  In this case, it should subscribe
// to both the base and experiment drivers, because we might get messages for
// either.
func (f *flaggedDriver) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	out := make(chan *message.Message)
	base, err := f.baseSub.Subscribe(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("Failed to subscribe to base: %w", err)
	}
	experiment, err := f.experimentSub.Subscribe(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("Failed to subscribe to experiment: %w", err)
	}
	go func() {
		defer close(out)
		// Cribbed from https://medium.com/justforfunc/why-are-there-nil-channels-in-go-9877cc0b2308
		for base != nil || experiment != nil {
			select {
			case msg, ok := <-base:
				if !ok {
					base = nil
					continue
				}
				out <- msg
				f.readMessages.Add(ctx, 1, metric.WithAttributes(attribute.Bool("experiment", false)))
			case msg, ok := <-experiment:
				if !ok {
					experiment = nil
					continue
				}
				out <- msg
				f.readMessages.Add(ctx, 1, metric.WithAttributes(attribute.Bool("experiment", true)))
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

// Close implements message.Publisher and message.Subscriber.  It closes all
// the drivers managed by the flagged publisher.
func (f *flaggedDriver) Close() error {
	if err := f.basePub.Close(); err != nil {
		return fmt.Errorf("Failed to close base publisher: %w", err)
	}
	if err := f.experimentPub.Close(); err != nil {
		return fmt.Errorf("Failed to close experiment publisher: %w", err)
	}
	return nil
}

func makeFlaggedDriver(ctx context.Context, cfg *serverconfig.EventConfig, flagClient openfeature.IClient,
) (message.Publisher, message.Subscriber, common.DriverCloser, error) {
	meter := otel.Meter(metricsSubsystem)
	publishedMessages, err := meter.Int64Counter("events_published")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to create published messages counter: %w", err)
	}
	readMessages, err := meter.Int64Counter("events_read")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to create read messages counter: %w", err)
	}

	basePub, baseSub, baseCloser, err := instantiateDriver(ctx, cfg.Flags.MainDriver, cfg, flagClient)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to instantiate base: %w", err)
	}
	experimentPub, experimentSub, experimentCloser, err := instantiateDriver(ctx, cfg.Flags.AlternateDriver, cfg, flagClient)
	if err != nil {
		baseCloser()
		return nil, nil, nil, fmt.Errorf("Failed to instantiate experiment: %w", err)
	}

	ret := &flaggedDriver{
		basePub:       basePub,
		experimentPub: experimentPub,
		baseSub:       baseSub,
		experimentSub: experimentSub,

		flags:             flagClient,
		publishedMessages: publishedMessages,
		readMessages:      readMessages,
	}
	closer := func() {
		baseCloser()
		experimentCloser()
	}

	return ret, ret, closer, nil
}

var _ message.Publisher = (*flaggedDriver)(nil)
var _ message.Subscriber = (*flaggedDriver)(nil)
