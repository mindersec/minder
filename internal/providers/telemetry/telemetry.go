// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/puzpuzpuz/xsync/v3"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/mindersec/minder/internal/db"
)

var _ http.RoundTripper = (*instrumentedRoundTripper)(nil)

type instrumentedRoundTripper struct {
	baseRoundTripper            http.RoundTripper
	durationHistogram           metric.Int64Histogram
	rateLimitRemainingHistogram metric.Int64Histogram
	throttledCounter            metric.Int64Counter
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

	if resp != nil {
		irt.recordRateLimitMetrics(r.Context(), resp, labels)
	}

	return resp, err
}

func (irt *instrumentedRoundTripper) recordRateLimitMetrics(
	ctx context.Context,
	resp *http.Response,
	labels []attribute.KeyValue,
) {
	remainingHeader := resp.Header.Get("X-RateLimit-Remaining")
	// Parse the header once; use math.MinInt64 as sentinel when absent or unparseable.
	remaining := int64(math.MinInt64)
	if remainingHeader != "" {
		if parsed, err := strconv.ParseInt(remainingHeader, 10, 64); err == nil {
			remaining = parsed
		}
	}

	if remaining != math.MinInt64 {
		irt.rateLimitRemainingHistogram.Record(ctx, remaining, metric.WithAttributes(labels...))
	}

	// Check for throttling (403 or 429 with rate limit headers)
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		// A 429 is always throttled; a 403 is throttled only when quota is exhausted.
		isThrottled := resp.StatusCode == http.StatusTooManyRequests
		// then add the case where we don't get a 429, but we have 0 quota remaining
		if !isThrottled && remaining == 0 {
			isThrottled = true
		}

		if isThrottled {
			irt.throttledCounter.Add(ctx, 1, metric.WithAttributes(labels...))
		}
	}
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

	httpProviderHistograms       *xsync.MapOf[db.ProviderType, metric.Int64Histogram]
	rateLimitRemainingHistograms *xsync.MapOf[db.ProviderType, metric.Int64Histogram]
	throttledCounters            *xsync.MapOf[db.ProviderType, metric.Int64Counter]
}

func newProviderMapOf[V any]() *xsync.MapOf[db.ProviderType, V] {
	return xsync.NewMapOf[db.ProviderType, V]()
}

// newHttpClientMetrics creates a new http provider metrics instance.
func newHttpClientMetrics() *httpClientMetrics {
	return &httpClientMetrics{
		providersMeter:               otel.Meter("providers"),
		httpProviderHistograms:       newProviderMapOf[metric.Int64Histogram](),
		rateLimitRemainingHistograms: newProviderMapOf[metric.Int64Histogram](),
		throttledCounters:            newProviderMapOf[metric.Int64Counter](),
	}
}

func (m *httpClientMetrics) createProviderHistogram(providerType db.ProviderType) (metric.Int64Histogram, error) {
	histogramName := fmt.Sprintf("%s.http.roundtrip.duration", providerType)
	return m.providersMeter.Int64Histogram(histogramName,
		metric.WithDescription("HTTP roundtrip duration for provider"),
		metric.WithUnit("ms"),
	)
}

func (m *httpClientMetrics) createRateLimitRemainingHistogram(providerType db.ProviderType) (metric.Int64Histogram, error) {
	histogramName := fmt.Sprintf("%s.http.ratelimit.remaining", providerType)
	return m.providersMeter.Int64Histogram(histogramName,
		metric.WithDescription("HTTP rate limit remaining for provider"),
		metric.WithUnit("1"),
	)
}

func (m *httpClientMetrics) createThrottledCounter(providerType db.ProviderType) (metric.Int64Counter, error) {
	counterName := fmt.Sprintf("%s.http.throttled.count", providerType)
	return m.providersMeter.Int64Counter(counterName,
		metric.WithDescription("Count of throttled HTTP requests for provider"),
		metric.WithUnit("1"),
	)
}

func (m *httpClientMetrics) getHistogramForProvider(providerType db.ProviderType) metric.Int64Histogram {
	histogram, _ := m.httpProviderHistograms.LoadOrCompute(providerType, func() metric.Int64Histogram {
		newHistogram, err := m.createProviderHistogram(providerType)
		if err != nil {
			log.Printf("failed to create duration histogram for provider %s: %v", providerType, err)
			return nil
		}
		return newHistogram
	})
	return histogram
}

func (m *httpClientMetrics) getRateLimitHistogramForProvider(providerType db.ProviderType) metric.Int64Histogram {
	histogram, _ := m.rateLimitRemainingHistograms.LoadOrCompute(providerType, func() metric.Int64Histogram {
		newHistogram, err := m.createRateLimitRemainingHistogram(providerType)
		if err != nil {
			log.Printf("failed to create rate limit histogram for provider %s: %v", providerType, err)
			return nil
		}
		return newHistogram
	})
	return histogram
}

func (m *httpClientMetrics) getThrottledCounterForProvider(providerType db.ProviderType) metric.Int64Counter {
	counter, _ := m.throttledCounters.LoadOrCompute(providerType, func() metric.Int64Counter {
		newCounter, err := m.createThrottledCounter(providerType)
		if err != nil {
			log.Printf("failed to create throttled counter for provider %s: %v", providerType, err)
			return nil
		}
		return newCounter
	})
	return counter
}

func (m *httpClientMetrics) NewDurationRoundTripper(
	wrapped http.RoundTripper,
	providerType db.ProviderType,
) (http.RoundTripper, error) {
	histogram := m.getHistogramForProvider(providerType)
	if histogram == nil {
		return nil, fmt.Errorf("failed to retrieve duration histogram for provider %s", providerType)
	}

	rateLimitRemainingHistogram := m.getRateLimitHistogramForProvider(providerType)
	if rateLimitRemainingHistogram == nil {
		return nil, fmt.Errorf("failed to retrieve rate limit histogram for provider %s", providerType)
	}

	throttledCounter := m.getThrottledCounterForProvider(providerType)
	if throttledCounter == nil {
		return nil, fmt.Errorf("failed to retrieve throttled counter for provider %s", providerType)
	}

	return &instrumentedRoundTripper{
		baseRoundTripper:            wrapped,
		durationHistogram:           histogram,
		rateLimitRemainingHistogram: rateLimitRemainingHistogram,
		throttledCounter:            throttledCounter,
	}, nil
}
