// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog"
)

type key int

const (
	// telemetryContextKey is a key used to store the current telemetry record's context
	telemetryContextKey key = iota
)

// TelemetryStore is a struct that can be used to store telemetry data in the context.
type TelemetryStore struct {
	data map[string]any
}

// BusinessRecord provides the ability to store an observation about the current
// flow of business logic in the context of the current request.  When called in
// in the context of a logged action, it will record and send the marshalled data
// to the logging system.
//
// When called outside a logged context, it will collect and discard the data.
func BusinessRecord(ctx context.Context) map[string]any {
	ts, ok := ctx.Value(telemetryContextKey).(*TelemetryStore)
	if !ok {
		// return a dummy value
		return make(map[string]any)
	}
	// Intentionally allowing aliasing here, we want to collect all data for one
	// from different execution points, then write it out at completion.
	return ts.data
}

// ================== ALTERNATE IMPLEMENTATION OPTION ==================

// Recordable represents a simple type that we can record for telemetry.
type Recordable interface {
	int | int32 | int64 | float32 | float64 | ~string | bool | time.Time | time.Duration
}

// BusinessRecord2 provides the ability to store simple values.
// We can't add json.Marshaller to the Recordable type, so there's
// also BusinessRecord3 for complex types.
func BusinessRecord2[T Recordable](ctx context.Context, key string, data T) error {
	ts, ok := ctx.Value(telemetryContextKey).(*TelemetryStore)
	if !ok {
		// No telemetry in context -- allow to pass through
		return nil
	}
	if ts.data[key] != nil {
		return fmt.Errorf("key %q already recorded", key)
	}
	ts.data[key] = data
	return nil
}

// BusinessRecord3 provides the ability to store complex data that can be marshalled as JSON.
func BusinessRecord3(ctx context.Context, key string, data json.Marshaler) error {
	ts, ok := ctx.Value(telemetryContextKey).(*TelemetryStore)
	if !ok {
		// No telemetry in context -- allow to pass through
		return nil
	}
	if ts.data[key] != nil {
		return fmt.Errorf("key %q already recorded", key)
	}
	ts.data[key] = data
	return nil
}

// WithTelemetry enriches the current context with a TelemetryStore which will
// collect observations about the current flow of business logic.
func (ts *TelemetryStore) WithTelemetry(ctx context.Context) context.Context {
	if ts == nil {
		return ctx
	}
	// Initialize the map if it doesn't exist
	if ts.data == nil {
		ts.data = make(map[string]any)
	}
	return context.WithValue(ctx, telemetryContextKey, ts)
}

// Record adds the collected data to the supplied event record.
func (ts *TelemetryStore) Record(e *zerolog.Event) *zerolog.Event {
	if ts == nil || ts.data == nil {
		return e
	}
	e.Fields(ts.data)
	e.Bool("telemetry", true)
	// Note: we explicitly don't call e.Send() here so that Send() occurs in the
	// same scope as the event is created.
	return e
}
