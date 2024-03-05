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

	"github.com/stacklok/minder/internal/db"
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

	httpProviderHistograms *xsync.MapOf[db.ProviderTrait, metric.Int64Histogram]
}

func newProviderMapOf[V any]() *xsync.MapOf[db.ProviderTrait, V] {
	return xsync.NewMapOf[db.ProviderTrait, V]()
}

// newHttpClientMetrics creates a new http provider metrics instance.
func newHttpClientMetrics() *httpClientMetrics {
	return &httpClientMetrics{
		providersMeter:         otel.Meter("providers"),
		httpProviderHistograms: newProviderMapOf[metric.Int64Histogram](),
	}
}

func (m *httpClientMetrics) createProviderHistogram(providerType db.ProviderTrait) (metric.Int64Histogram, error) {
	histogramName := fmt.Sprintf("%s.http.roundtrip.duration", providerType)
	return m.providersMeter.Int64Histogram(histogramName,
		metric.WithDescription("HTTP roundtrip duration for provider"),
		metric.WithUnit("ms"),
	)
}

func (m *httpClientMetrics) getHistogramForProvider(providerType db.ProviderTrait) metric.Int64Histogram {
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
	providerType db.ProviderTrait,
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
