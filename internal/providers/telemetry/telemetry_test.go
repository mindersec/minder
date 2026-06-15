// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/mindersec/minder/internal/db"
)

type mockRoundTripper struct {
	resp *http.Response
	err  error
}

func (m *mockRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return m.resp, m.err
}

func TestInstrumentedRoundTripper_RateLimitMetrics(t *testing.T) {
	t.Parallel()
	// Initialize OTel metrics for testing
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	metrics := &httpClientMetrics{
		providersMeter:               mp.Meter("test"),
		httpProviderHistograms:       newProviderMapOf[metric.Int64Histogram](),
		rateLimitRemainingHistograms: newProviderMapOf[metric.Int64Histogram](),
		throttledCounters:            newProviderMapOf[metric.Int64Counter](),
	}

	providerType := db.ProviderTypeGithub

	tests := []struct {
		name            string
		statusCode      int
		headers         map[string]string
		expectRemaining int64
		expectThrottled bool
	}{
		{
			name:       "normal request",
			statusCode: 200,
			headers: map[string]string{
				"X-RateLimit-Remaining": "4500",
			},
			expectRemaining: 4500,
			expectThrottled: false,
		},
		{
			name:       "throttled request (403 and remaining 0)",
			statusCode: 403,
			headers: map[string]string{
				"X-RateLimit-Remaining": "0",
			},
			expectRemaining: 0,
			expectThrottled: true,
		},
		{
			name:       "throttled request (429)",
			statusCode: 429,
			headers: map[string]string{
				"X-RateLimit-Remaining": "0",
			},
			expectRemaining: 0,
			expectThrottled: true,
		},
		{
			name:       "not throttled (403 but remaining > 0)",
			statusCode: 403,
			headers: map[string]string{
				"X-RateLimit-Remaining": "100",
			},
			expectRemaining: 100,
			expectThrottled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Header:     make(http.Header),
				Request:    httptest.NewRequest("GET", "https://api.github.com/repos/test/repo", nil),
			}
			for k, v := range tt.headers {
				resp.Header.Set(k, v)
			}

			mockRT := &mockRoundTripper{resp: resp}
			irt, err := metrics.NewDurationRoundTripper(mockRT, providerType)
			require.NoError(t, err)

			_, err = irt.RoundTrip(resp.Request)
			require.NoError(t, err)

			// Collect metrics
			var rm metricdata.ResourceMetrics
			err = reader.Collect(context.Background(), &rm)
			require.NoError(t, err)

			// Verify metrics
			foundRemaining := false
			foundThrottled := false

			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					if m.Name == "github.http.ratelimit.remaining" {
						data := m.Data.(metricdata.Histogram[int64])
						// Check the most recent record
						assert.Greater(t, data.DataPoints[0].Count, uint64(0))
						// In a histogram it's harder to check the exact value without more complex matching,
						// but we can check if it was recorded.
						foundRemaining = true
					}
					if m.Name == "github.http.throttled.count" && tt.expectThrottled {
						data := m.Data.(metricdata.Sum[int64])
						assert.Equal(t, int64(1), data.DataPoints[0].Value)
						foundThrottled = true
					}
				}
			}

			assert.True(t, foundRemaining, "RateLimit-Remaining metric not found")
			if tt.expectThrottled {
				assert.True(t, foundThrottled, "Throttled count metric not found")
			}
		})
	}
}
