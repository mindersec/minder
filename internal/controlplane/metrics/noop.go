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

// Package metrics defines the primitives available for the controlplane metrics
package metrics

import (
	"context"

	"github.com/stacklok/minder/internal/db"
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
