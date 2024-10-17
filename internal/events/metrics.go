// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func recordMetrics(instruments *messageInstruments) func(h message.HandlerFunc) message.HandlerFunc {
	metricsFunc := func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			var processingTime time.Duration
			if publishedAt := msg.Metadata.Get(PublishedKey); publishedAt != "" {
				if parsedTime, err := time.Parse(time.RFC3339, publishedAt); err == nil {
					processingTime = time.Since(parsedTime)
				}
			}

			res, err := h(msg)

			// Defer the DLQ tracking logic to after the message has been processed by other middlewares,
			// including the deferred PoisonQueue middleware functionality,
			// so that we can check if it has been poisoned or not.
			isPoisoned := msg.Metadata.Get(middleware.ReasonForPoisonedKey) != ""
			instruments.messageProcessingTimeHistogram.Record(
				msg.Context(),
				processingTime.Milliseconds(),
				metric.WithAttributes(attribute.Bool("poison", isPoisoned)),
			)

			return res, err
		}
	}
	return metricsFunc
}

func initMetricsInstruments(meter metric.Meter) (*messageInstruments, error) {
	histogram, err := createProcessingLatencyHistogram(meter)
	if err != nil {
		return nil, err
	}

	return &messageInstruments{
		messageProcessingTimeHistogram: histogram,
	}, nil
}

func createProcessingLatencyHistogram(meter metric.Meter) (metric.Int64Histogram, error) {
	processingLatencyHistogram, err := meter.Int64Histogram("messages.processing_delay",
		metric.WithDescription("Duration between a message being enqueued and dequeued for processing"),
		metric.WithUnit("ms"),
		// Pick a set of bucket boundaries that span out to 10 minutes
		metric.WithExplicitBucketBoundaries(0, 500, 1000, 2000, 5000, 10000, 30000, 60000, 120000, 300000, 600000),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create message processing processingLatencyHistogram: %w", err)
	}
	return processingLatencyHistogram, nil
}
