// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"fmt"
	"net/http"
	"time"

	"github.com/puzpuzpuz/xsync/v3"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/mindersec/minder/pkg/db"
)

var _ http.RoundTripper = (*instrumentedRoundTripper)(nil)

type instrumentedRoundTripper struct {
	baseRoundTripper  http.RoundTripper
	durationHistogram metric.Int64Histogram
}

func (irt *instrumentedRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	startTime := time.Now()

	resp, err := irt.baseRoundTripper.RoundTrip(r)

	duration := time.Since(startTime).Milliseconds()
	labels := []attribute.KeyValue{
		attribute.String("http_method", r.Method),
		attribute.String("http_host", r.URL.Host),
	}

	if resp != nil {
		labels = append(labels, attribute.Int("http_status_code", resp.StatusCode))
	}

	irt.durationHistogram.Record(r.Context(), duration, metric.WithAttributes(labels...))

	return resp, err
}

var _ ProviderMetrics = (*providerMetrics)(nil)

type providerMetrics struct {
	httpClientMetrics
}

// NewProviderMetrics creates a new provider metrics instance.
func NewProviderMetrics() *providerMetrics {
	return &providerMetrics{
		httpClientMetrics: *newHttpClientMetrics(),
	}
}

var _ HttpClientMetrics = (*httpClientMetrics)(nil)

type httpClientMetrics struct {
	providersMeter metric.Meter

	httpProviderHistograms *xsync.MapOf[db.ProviderType, metric.Int64Histogram]
}

func newProviderMapOf[V any]() *xsync.MapOf[db.ProviderType, V] {
	return xsync.NewMapOf[db.ProviderType, V]()
}

// newHttpClientMetrics creates a new http provider metrics instance.
func newHttpClientMetrics() *httpClientMetrics {
	return &httpClientMetrics{
		providersMeter:         otel.Meter("providers"),
		httpProviderHistograms: newProviderMapOf[metric.Int64Histogram](),
	}
}

func (m *httpClientMetrics) createProviderHistogram(providerType db.ProviderType) (metric.Int64Histogram, error) {
	histogramName := fmt.Sprintf("%s.http.roundtrip.duration", providerType)
	return m.providersMeter.Int64Histogram(histogramName,
		metric.WithDescription("HTTP roundtrip duration for provider"),
		metric.WithUnit("ms"),
	)
}

func (m *httpClientMetrics) getHistogramForProvider(providerType db.ProviderType) metric.Int64Histogram {
	histogram, _ := m.httpProviderHistograms.LoadOrCompute(providerType, func() metric.Int64Histogram {
		newHistogram, err := m.createProviderHistogram(providerType)
		if err != nil {
			log.Printf("failed to create histogram for provider %s: %v", providerType, err)
			return nil
		}
		return newHistogram
	})
	return histogram
}

func (m *httpClientMetrics) NewDurationRoundTripper(
	wrapped http.RoundTripper,
	providerType db.ProviderType,
) (http.RoundTripper, error) {
	histogram := m.getHistogramForProvider(providerType)
	if histogram == nil {
		return nil, fmt.Errorf("failed to retrieve histogram for provider %s", providerType)
	}

	return &instrumentedRoundTripper{
		baseRoundTripper:  wrapped,
		durationHistogram: histogram,
	}, nil
}
