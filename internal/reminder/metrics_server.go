// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package reminder

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	metricsPath       = "/metrics"
	readHeaderTimeout = 2 * time.Second
)

func (r *reminder) startMetricServer(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)

	prometheusExporter, err := prometheus.New(
		prometheus.WithNamespace("reminder"),
	)
	if err != nil {
		return fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("reminder"),
		// TODO: Make this auto-generated
		semconv.ServiceVersion("v0.1.0"),
	)

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(prometheusExporter),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(mp)

	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.Handler())

	server := &http.Server{
		Addr:              r.cfg.MetricServer.GetAddress(),
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	logger.Info().Msgf("starting metrics server on %s", server.Addr)

	// Start the metrics server
	go func() {
		err = server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Err(err).Msg("error starting metrics server")
		}
	}()

	// Watch for context cancellation or stop signal to shutdown the metrics server
	go func() {
		select {
		case <-ctx.Done():
		case <-r.stop:
		}

		// shutdown the metrics server when either the context is done or when reminder is stopped
		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownRelease()

		logger.Info().Msg("shutting down metrics server")

		if err = server.Shutdown(shutdownCtx); err != nil {
			logger.Err(err).Msg("error shutting down metrics server")
		}

		if err = mp.Shutdown(shutdownCtx); err != nil {
			logger.Err(err).Msg("error shutting down metrics provider")
		}

		close(r.metricsServerDone)
	}()

	return nil
}
