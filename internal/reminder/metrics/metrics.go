// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package metrics provides metrics for the reminder service
package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

// Default bucket boundaries in seconds for the delay histograms
var delayBuckets = []float64{
	60,    // 1 minute
	300,   // 5 minutes
	600,   // 10 minutes
	1800,  // 30 minutes
	3600,  // 1 hour
	7200,  // 2 hours
	10800, // 3 hours
	18000, // 5 hours
	25200, // 7 hours
	36000, // 10 hours
}

// Metrics contains all the metrics for the reminder service
type Metrics struct {
	// Time between when a reminder became eligible and when it was sent
	SendDelay metric.Float64Histogram

	// Time between when a reminder became eligible and when it was sent for the first time
	NewSendDelay metric.Float64Histogram

	// Current number of reminders in the batch
	BatchSize metric.Int64Histogram
}

// NewMetrics creates a new metrics instance
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	sendDelay, err := meter.Float64Histogram(
		"send_delay",
		metric.WithDescription("Time between reminder becoming eligible and actual send (seconds)"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(delayBuckets...),
	)
	if err != nil {
		return nil, err
	}

	newSendDelay, err := meter.Float64Histogram(
		"new_send_delay",
		metric.WithDescription("Time between reminder becoming eligible and actual send (seconds) for first time reminders"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(delayBuckets...),
	)
	if err != nil {
		return nil, err
	}

	batchSize, err := meter.Int64Histogram(
		"batch_size",
		metric.WithDescription("Current number of reminders in the batch"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		SendDelay:    sendDelay,
		NewSendDelay: newSendDelay,
		BatchSize:    batchSize,
	}, nil
}

// RecordBatch records the metrics for a batch of reminders
func (m *Metrics) RecordBatch(ctx context.Context, size int64) {
	m.BatchSize.Record(ctx, size)
}
