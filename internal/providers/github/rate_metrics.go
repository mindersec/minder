// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"sync"

	"github.com/google/go-github/v63/github"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

const (
	rateLimitRemainingMetric = "github.api.ratelimit.remaining"
	rateLimitLimitMetric     = "github.api.ratelimit.limit"
)

var (
	rateMetricsInit sync.Once
	remainingGauge  metric.Int64Gauge
	limitGauge      metric.Int64Gauge
)

func initRateMetrics() {
	rateMetricsInit.Do(func() {
		meter := otel.Meter("github")
		var err error
		remainingGauge, err = meter.Int64Gauge(
			rateLimitRemainingMetric,
			metric.WithDescription("Remaining GitHub API rate limit from github.Response.Rate"),
			metric.WithUnit("1"),
		)
		if err != nil {
			zerolog.Ctx(context.Background()).Warn().Err(err).Msg("creating GitHub remaining rate-limit gauge failed")
		}
		limitGauge, err = meter.Int64Gauge(
			rateLimitLimitMetric,
			metric.WithDescription("GitHub API rate limit from github.Response.Rate"),
			metric.WithUnit("1"),
		)
		if err != nil {
			zerolog.Ctx(context.Background()).Warn().Err(err).Msg("creating GitHub rate-limit gauge failed")
		}
	})
}

// ObserveResponseRate records remaining and limit from a GitHub API response
// before the provider returns that response to callers. A nil response or an
// unset Rate (Limit 0) is ignored.
func ObserveResponseRate(ctx context.Context, resp *github.Response) {
	initRateMetrics()
	observeResponseRate(ctx, remainingGauge, limitGauge, resp)
}

func observeResponseRate(
	ctx context.Context,
	remaining metric.Int64Gauge,
	limit metric.Int64Gauge,
	resp *github.Response,
) {
	if resp == nil || resp.Rate.Limit == 0 {
		return
	}
	if remaining != nil {
		remaining.Record(ctx, int64(resp.Rate.Remaining))
	}
	if limit != nil {
		limit.Record(ctx, int64(resp.Rate.Limit))
	}
}
