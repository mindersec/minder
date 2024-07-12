// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package engine

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/metrics/meters"
)

// ExecutorMetrics encapsulates metrics operations for the executor
type ExecutorMetrics struct {
	evalCounter        metric.Int64Counter
	remediationCounter metric.Int64Counter
	alertCounter       metric.Int64Counter
	entityDuration     metric.Int64Histogram
	profileDuration    metric.Int64Histogram
}

// NewExecutorMetrics instantiates the ExecutorMetrics struct.
func NewExecutorMetrics(meterFactory meters.MeterFactory) (*ExecutorMetrics, error) {
	meter := meterFactory.Build("executor")
	evalCounter, err := meter.Int64Counter("eval.status",
		metric.WithDescription("Number of rule evaluation statuses"),
		metric.WithUnit("evaluations"))
	if err != nil {
		return nil, fmt.Errorf("failed to create eval counter: %w", err)
	}

	remediationCounter, err := meter.Int64Counter("eval.remediation",
		metric.WithDescription("Number of remediation statuses"),
		metric.WithUnit("evaluations"))
	if err != nil {
		return nil, fmt.Errorf("failed to create remediation counter: %w", err)
	}

	alertCounter, err := meter.Int64Counter("eval.alert",
		metric.WithDescription("Number of alert statuses"),
		metric.WithUnit("evaluations"))
	if err != nil {
		return nil, fmt.Errorf("failed to create alert counter: %w", err)
	}

	profileDuration, err := meter.Int64Histogram("eval.entity-eval-duration",
		metric.WithDescription("Time taken to evaluate all profiles against an entity"),
		metric.WithUnit("milliseconds"))
	if err != nil {
		return nil, fmt.Errorf("failed to create profile histogram: %w", err)
	}

	entityDuration, err := meter.Int64Histogram("eval.profile-eval-duration",
		metric.WithDescription("Time taken to evaluate a single profile against an entity"),
		metric.WithUnit("milliseconds"))
	if err != nil {
		return nil, fmt.Errorf("failed to create entity histogram: %w", err)
	}

	return &ExecutorMetrics{
		evalCounter:        evalCounter,
		remediationCounter: remediationCounter,
		alertCounter:       alertCounter,
		profileDuration:    profileDuration,
		entityDuration:     entityDuration,
	}, nil
}

// CountEvalStatus counts evaluation events by status.
func (e *ExecutorMetrics) CountEvalStatus(
	ctx context.Context,
	status db.EvalStatusTypes,
	entityType db.Entities,
) {
	e.evalCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("entity_type", string(entityType)),
		attribute.String("status", string(status)),
	))
}

// CountRemediationStatus counts remediation events by status.
func (e *ExecutorMetrics) CountRemediationStatus(
	ctx context.Context,
	status db.RemediationStatusTypes,
) {
	e.evalCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", string(status)),
	))
}

// CountAlertStatus counts alert events by status.
func (e *ExecutorMetrics) CountAlertStatus(
	ctx context.Context,
	status db.AlertStatusTypes,
) {
	e.evalCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", string(status)),
	))
}

// TimeEntityEvaluation records how long it took to evaluate a profile.
func (e *ExecutorMetrics) TimeEntityEvaluation(ctx context.Context, startTime time.Time) {
	e.entityDuration.Record(ctx, time.Since(startTime).Milliseconds())
}

// TimeProfileEvaluation records how long it took to evaluate a profile.
func (e *ExecutorMetrics) TimeProfileEvaluation(ctx context.Context, startTime time.Time) {
	e.profileDuration.Record(ctx, time.Since(startTime).Milliseconds())
}
