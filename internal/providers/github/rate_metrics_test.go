// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"testing"

	"github.com/google/go-github/v63/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestObserveResponseRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		resp            *github.Response
		expectRemaining *int64
		expectLimit     *int64
	}{
		{
			name: "populated rate",
			resp: &github.Response{
				Rate: github.Rate{Remaining: 4500, Limit: 5000},
			},
			expectRemaining: int64Ptr(4500),
			expectLimit:     int64Ptr(5000),
		},
		{
			name:            "nil response",
			resp:            nil,
			expectRemaining: nil,
			expectLimit:     nil,
		},
		{
			name:            "unset rate",
			resp:            &github.Response{},
			expectRemaining: nil,
			expectLimit:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader := sdkmetric.NewManualReader()
			mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
			t.Cleanup(func() {
				require.NoError(t, mp.Shutdown(context.Background()))
			})

			meter := mp.Meter("github")
			remaining, err := meter.Int64Gauge(rateLimitRemainingMetric)
			require.NoError(t, err)
			limit, err := meter.Int64Gauge(rateLimitLimitMetric)
			require.NoError(t, err)

			observeResponseRate(context.Background(), remaining, limit, tt.resp)

			var rm metricdata.ResourceMetrics
			require.NoError(t, reader.Collect(context.Background(), &rm))

			gotRemaining, gotLimit := gaugeValues(t, rm)
			if tt.expectRemaining == nil {
				assert.Nil(t, gotRemaining, "remaining gauge should not be recorded")
			} else {
				require.NotNil(t, gotRemaining)
				assert.Equal(t, *tt.expectRemaining, *gotRemaining)
			}
			if tt.expectLimit == nil {
				assert.Nil(t, gotLimit, "limit gauge should not be recorded")
			} else {
				require.NotNil(t, gotLimit)
				assert.Equal(t, *tt.expectLimit, *gotLimit)
			}
		})
	}
}

func int64Ptr(v int64) *int64 {
	return &v
}

func gaugeValues(t *testing.T, rm metricdata.ResourceMetrics) (remaining *int64, limit *int64) {
	t.Helper()
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			data, ok := m.Data.(metricdata.Gauge[int64])
			if !ok || len(data.DataPoints) == 0 {
				continue
			}
			val := data.DataPoints[0].Value
			switch m.Name {
			case rateLimitRemainingMetric:
				remaining = &val
			case rateLimitLimitMetric:
				limit = &val
			}
		}
	}
	return remaining, limit
}

// Ensure the public helper does not panic when gauges are uninitialized in tests.
func TestObserveResponseRatePublicNilSafe(t *testing.T) {
	t.Parallel()
	ObserveResponseRate(context.Background(), nil)
	ObserveResponseRate(context.Background(), &github.Response{})
}
