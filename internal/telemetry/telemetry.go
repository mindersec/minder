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

// Package telemetry provides the telemetry object which is responsible for manual instrumentation of metrics
package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// Telemetry provides the functions for recording telemetry
type Telemetry interface {
	UserRegistered(ctx context.Context)
}

// OtelTelemetry is a wrapper over the open telemetry meter instruments
type OtelTelemetry struct {
	userCounter *metric.Int64Counter
}

// NewOtelTelemetry creates an OtelTelemetry object for manual instrumentation
func NewOtelTelemetry() (Telemetry, error) {
	meter := otel.Meter("controlplane")
	registrationCounter, err := meter.Int64Counter(
		"user.registrations",
		metric.WithDescription("Number of user registrations."),
		metric.WithUnit("{registration}"),
	)
	if err != nil {
		return nil, err
	}
	return &OtelTelemetry{
		userCounter: &registrationCounter,
	}, nil
}

// UserRegistered increments the user counter if enabled
func (t *OtelTelemetry) UserRegistered(ctx context.Context) {
	if t.userCounter != nil {
		(*t.userCounter).Add(ctx, 1)
	}
}
