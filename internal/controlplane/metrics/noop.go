// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package metrics defines the primitives available for the controlplane metrics
package metrics

import (
	"context"

	"github.com/mindersec/minder/pkg/db"
)

type noopMetrics struct{}

// NewNoopMetrics creates a new controlplane metrics instance.
func NewNoopMetrics() Metrics {
	return &noopMetrics{}
}

// Init implements Metrics.Init
func (_ *noopMetrics) Init(_ db.Store) error {
	return nil
}

// AddWebhookEventTypeCount implements Metrics.AddWebhookEventTypeCount
func (_ *noopMetrics) AddWebhookEventTypeCount(_ context.Context, _ *WebhookEventState) {}

// AddTokenOpCount implements Metrics.AddTokenOpCount
func (_ *noopMetrics) AddTokenOpCount(_ context.Context, _ string, _ bool) {}
