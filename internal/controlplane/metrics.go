//
// Copyright 2023 Stacklok, Inc.
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

package controlplane

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/stacklok/mediator/internal/db"
)

type metrics struct {
	meter metric.Meter

	// webhook http codes by type
	webhookStatusCodeCounter metric.Int64Counter
	// webhook event type counter
	webhookEventTypeCounter metric.Int64Counter
}

// NewMetrics creates a new controlplane metrics instance.
func NewMetrics() *metrics {
	return &metrics{
		meter: otel.Meter("controlplane"),
	}
}

func (m *metrics) initInstruments(store db.Store) error {
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

	return nil
}

type webhookEventState struct {
	// the type of the event, e.g. pull_request, repository, workflow_run, ...
	typ string
	// whether the event was accepted by engine or filtered out
	accepted bool
	// whether there was an error processing the event
	error bool
}

func (m *metrics) webhookEventTypeCount(ctx context.Context, state webhookEventState) {
	if m.webhookEventTypeCounter == nil {
		return
	}

	labels := []attribute.KeyValue{
		attribute.String("webhook_event.type", state.typ),
		attribute.Bool("webhook_event.accepted", state.accepted),
		attribute.Bool("webhook_event.error", state.error),
	}
	m.webhookEventTypeCounter.Add(ctx, 1, metric.WithAttributes(labels...))
}
