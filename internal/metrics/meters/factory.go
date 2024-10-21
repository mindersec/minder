// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package meters contains the OpenTelemetry meter factories.
package meters

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

// MeterFactory is an interface which hides the details of creating an
// OpenTelemetry metrics meter. This is used to select between a real exporter
// or noop for testing.
type MeterFactory interface {
	// Build creates a meter with the specified name.
	Build(name string) metric.Meter
}

// ExportingMeterFactory uses the "real" OpenTelemetry metric meter
type ExportingMeterFactory struct{}

// Build creates a meter with the specified name.
func (_ *ExportingMeterFactory) Build(name string) metric.Meter {
	return otel.Meter(name)
}

// NoopMeterFactory returns a noop metrics meter
type NoopMeterFactory struct{}

// Build returns a noop meter implementation.
func (_ *NoopMeterFactory) Build(_ string) metric.Meter {
	return noop.Meter{}
}
