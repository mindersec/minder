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
	"fmt"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/stacklok/minder/internal/db"
)

// WebhookEventState represents the state of a webhook event
type WebhookEventState struct {
	// Typ is the type of the event, e.g. pull_request, repository, workflow_run, ...
	Typ string
	// Accepted is whether the event was accepted by engine or filtered out
	Accepted bool
	// Error is whether there was an error processing the event
	Error bool
}

// Metrics implements metrics management for the control plane
type Metrics interface {
	// Initialize metrics engine
	Init(db.Store) error

	// AddWebhookEventTypeCount adds a count to the webhook event type counter
	AddWebhookEventTypeCount(context.Context, *WebhookEventState)
}

type metricsImpl struct {
	meter           metric.Meter
	instrumentsOnce sync.Once

	// webhook http codes by type
	webhookStatusCodeCounter metric.Int64Counter
	// webhook event type counter
	webhookEventTypeCounter metric.Int64Counter

	// Track how often users who register a token are correlated with the
	// GitHub user from GetAuthorizationURL
	tokenOpCounter metric.Int64Counter
}

// NewMetrics creates a new controlplane metrics instance.
func NewMetrics() Metrics {
	return &metricsImpl{
		meter: otel.Meter("controlplane"),
	}
}

// Init initializes the metrics engine
func (m *metricsImpl) Init(store db.Store) error {
	var err error
	m.instrumentsOnce.Do(func() {
		err = m.initInstrumentsOnce(store)
	})
	return err
}

func (m *metricsImpl) initInstrumentsOnce(store db.Store) error {
	_, err := m.meter.Int64ObservableGauge("user.count",
		metric.WithDescription("Number of users in the database"),
		metric.WithUnit("users"),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			c, err := store.CountUsers(ctx)
			if err != nil {
				return err
			}
			observer.Observe(c)
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create user count gauge: %w", err)
	}

	_, err = m.meter.Int64ObservableGauge("repository.count",
		metric.WithDescription("Number of repositories in the database"),
		metric.WithUnit("repositories"),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			c, err := store.CountRepositories(ctx)
			if err != nil {
				return err
			}
			observer.Observe(c)
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create repository count gauge: %w", err)
	}

	_, err = m.meter.Int64ObservableGauge("profile_entity.count",
		metric.WithDescription("Number of profiles in the database, labeled by entity type"),
		metric.WithUnit("profiles"),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			rows, err := store.CountProfilesByEntityType(ctx)
			if err != nil {
				return err
			}
			for _, row := range rows {
				labels := []attribute.KeyValue{
					attribute.String("entity_type", string(row.ProfileEntity)),
				}
				observer.Observe(row.NumProfiles, metric.WithAttributes(labels...))
			}
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create profile count gauge: %w", err)
	}

	_, err = m.meter.Int64ObservableGauge("profile.quickstart.count",
		metric.WithDescription("Number of quickstart profiles in the database"),
		metric.WithUnit("profiles"),
		metric.WithInt64Callback(func(ctx context.Context, observer metric.Int64Observer) error {
			// the profile name is currently hardcoded in cmd/cli/app/quickstart/embed/profile.yaml
			const quickstartProfileName = "quickstart-profile"

			num, err := store.CountProfilesByName(ctx, quickstartProfileName)
			if err != nil {
				return err
			}
			observer.Observe(num)
			return nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create the quickstart profile count gauge: %w", err)
	}

	m.webhookStatusCodeCounter, err = m.meter.Int64Counter("webhook.status_code",
		metric.WithDescription("Number of webhook requests by status code"),
		metric.WithUnit("requests"))
	if err != nil {
		return fmt.Errorf("failed to create webhook status code counter: %w", err)
	}

	m.webhookEventTypeCounter, err = m.meter.Int64Counter("webhook.event_type",
		metric.WithDescription("Number of webhook events by event type"),
		metric.WithUnit("events"))
	if err != nil {
		return fmt.Errorf("failed to create webhook event type counter: %w", err)
	}

	m.tokenOpCounter, err = m.meter.Int64Counter("token-checks",
		metric.WithDescription("Number of times token URLs are issued and consumed"),
		metric.WithUnit("ops"))
	if err != nil {
		return fmt.Errorf("failed to create token operations counter: %w", err)
	}

	return nil
}

// AddWebhookEventTypeCount adds a count to the webhook event type counter
func (m *metricsImpl) AddWebhookEventTypeCount(ctx context.Context, state *WebhookEventState) {
	if m.webhookEventTypeCounter == nil {
		return
	}

	labels := []attribute.KeyValue{
		attribute.String("webhook_event.type", state.Typ),
		attribute.Bool("webhook_event.accepted", state.Accepted),
		attribute.Bool("webhook_event.error", state.Error),
	}
	m.webhookEventTypeCounter.Add(ctx, 1, metric.WithAttributes(labels...))
}

func (m *metricsImpl) AddTokenOpCount(ctx, stage string, hasId bool) {
	if m.tokenOpCounter == nil {
		return
	}

	m.tokenOpCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("stage", op),
		attribute.Bool("has-id", hasId)))
}
