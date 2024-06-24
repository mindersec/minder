// Copyright 2024 Stacklok, Inc.
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
