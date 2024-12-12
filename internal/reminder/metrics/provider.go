package metrics

import (
	"context"
	"errors"
	"fmt"
	"github.com/mindersec/minder/pkg/config/reminder"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
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

// Provider manages the metrics server and OpenTelemetry setup
type Provider struct {
	server  *http.Server
	mp      *sdkmetric.MeterProvider
	metrics *Metrics
}

// NewProvider creates a new metrics provider
func NewProvider(cfg *reminder.MetricsConfig) (*Provider, error) {
	if cfg == nil {
		return nil, errors.New("metrics config is nil")
	}

	if !cfg.Enabled {
		return &Provider{}, nil
	}

	// Create Prometheus exporter
	prometheusExporter, err := prometheus.New(
		prometheus.WithNamespace("reminder_service"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// Create resource with service information
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("reminder-service"),
		semconv.ServiceVersion("v0.1.0"),
	)

	// Create meter provider
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(prometheusExporter),
		sdkmetric.WithResource(res),
	)

	// Set global meter provider
	otel.SetMeterProvider(mp)

	// Create metrics
	meter := mp.Meter("reminder-service")
	metrics, err := NewMetrics(meter)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics: %w", err)
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.Handler())

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	return &Provider{
		server:  server,
		mp:      mp,
		metrics: metrics,
	}, nil
}

// Start starts the metrics server if enabled
func (p *Provider) Start(ctx context.Context) error {
	if p.server == nil {
		return nil // Metrics disabled
	}

	errCh := make(chan error)
	go func() {
		log.Info().Str("address", p.server.Addr).Msg("Starting metrics server")
		if err := p.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("metrics server error: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return p.Shutdown(ctx)
	}
}

// Shutdown gracefully shuts down the metrics server
func (p *Provider) Shutdown(ctx context.Context) error {
	if p.server == nil {
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	log.Info().Msg("Shutting down metrics server")
	if err := p.mp.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Error shutting down meter provider")
	}

	return p.server.Shutdown(shutdownCtx)
}

// Metrics returns the metrics instance
func (p *Provider) Metrics() *Metrics {
	return p.metrics
}
