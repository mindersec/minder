package metrics

import (
	"context"
	"go.opentelemetry.io/otel/metric"
)

// Default bucket boundaries in seconds for the send delay histogram
var sendDelayBuckets = []float64{
	0,    // immediate
	10,   // 10 seconds
	20,   // 20 seconds
	40,   // 40 seconds
	80,   // 1m 20s
	160,  // 2m 40s
	320,  // 5m 20s
	640,  // 10m 40s
	1280, // 21m 20s
}

type Metrics struct {
	// Time between when a reminder became eligible and when it was sent
	SendDelay metric.Float64Histogram

	// Current number of reminders in the batch
	BatchSize metric.Int64Gauge

	// Average batch size (updated on each batch)
	AvgBatchSize metric.Float64Gauge

	// For tracking average calculation
	// TODO: consider persisting this to avoid reset on restart (maybe)
	totalBatches   int64
	totalReminders int64
}

func NewMetrics(meter metric.Meter) (*Metrics, error) {
	sendDelay, err := meter.Float64Histogram(
		"reminder_send_delay",
		metric.WithDescription("Time between reminder becoming eligible and actual send (seconds)"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(sendDelayBuckets...),
	)
	if err != nil {
		return nil, err
	}

	batchSize, err := meter.Int64Gauge(
		"reminder_batch_size",
		metric.WithDescription("Current number of reminders in the batch"),
	)
	if err != nil {
		return nil, err
	}

	avgBatchSize, err := meter.Float64Gauge(
		"reminder_avg_batch_size",
		metric.WithDescription("Average number of reminders per batch"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		SendDelay:    sendDelay,
		BatchSize:    batchSize,
		AvgBatchSize: avgBatchSize,
	}, nil
}

func (m *Metrics) RecordBatch(ctx context.Context, size int64) {
	// Update current batch size
	m.BatchSize.Record(ctx, size)

	// Update running average
	m.totalBatches++
	m.totalReminders += size
	avgSize := float64(m.totalReminders) / float64(m.totalBatches)
	m.AvgBatchSize.Record(ctx, avgSize)
}

func (m *Metrics) RecordSendDelay(ctx context.Context, delaySeconds float64) {
	m.SendDelay.Record(ctx, delaySeconds)
}
