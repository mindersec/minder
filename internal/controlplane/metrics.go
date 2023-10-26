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
	"github.com/stacklok/mediator/internal/db"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"net/http"
)

// webhook http codes by type
var webhookStatusCodeCounter metric.Int64Counter

func initInstruments(store db.Store) error {
	meter := otel.Meter("controlplane")
	_, err := meter.Int64ObservableGauge("user.count",
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
		return err
	}

	webhookStatusCodeCounter, err = meter.Int64Counter("webhook.status_code",
		metric.WithDescription("Number of webhook requests by status code"),
		metric.WithUnit("requests"))
	return nil
}

func webhookStatusCodeMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)

		statusCode := recorder.status
		labels := []attribute.KeyValue{
			attribute.Int("status_code", statusCode),
		}
		ctx := r.Context()

		if webhookStatusCodeCounter != nil {
			webhookStatusCodeCounter.Add(ctx, 1, metric.WithAttributes(labels...))
		}
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
