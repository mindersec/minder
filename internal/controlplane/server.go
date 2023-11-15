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
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "github.com/signalfx/splunk-otel-go/instrumentation/github.com/lib/pq/splunkpq" // Auto-instrumented version of lib/pq
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/stacklok/minder/internal/assets"
	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/config"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/logger"
	provtelemetry "github.com/stacklok/minder/internal/providers/telemetry"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const metricsPath = "/metrics"

var (
	readHeaderTimeout = 2 * time.Second
)

// Server represents the controlplane server
type Server struct {
	store      db.Store
	cfg        *config.Config
	evt        *events.Eventer
	mt         *metrics
	provMt     provtelemetry.ProviderMetrics
	grpcServer *grpc.Server
	vldtr      auth.JwtValidator
	pb.UnimplementedHealthServiceServer
	pb.UnimplementedOAuthServiceServer
	pb.UnimplementedUserServiceServer
	pb.UnimplementedRepositoryServiceServer
	pb.UnimplementedProfileServiceServer
	pb.UnimplementedArtifactServiceServer
	pb.UnimplementedKeyServiceServer
	OAuth2       *oauth2.Config
	ClientID     string
	ClientSecret string
	cryptoEngine *crypto.Engine
}

// ServerOption is a function that modifies a server
type ServerOption func(*Server)

// WithProviderMetrics sets the provider metrics for the server
func WithProviderMetrics(mt provtelemetry.ProviderMetrics) ServerOption {
	return func(s *Server) {
		s.provMt = mt
	}
}

// NewServer creates a new server instance
func NewServer(
	store db.Store,
	evt *events.Eventer,
	cpm *metrics,
	cfg *config.Config,
	vldtr auth.JwtValidator,
	opts ...ServerOption,
) (*Server, error) {
	eng, err := crypto.EngineFromAuthConfig(&cfg.Auth)
	if err != nil {
		return nil, fmt.Errorf("failed to create crypto engine: %w", err)
	}
	s := &Server{
		store:        store,
		cfg:          cfg,
		evt:          evt,
		cryptoEngine: eng,
		vldtr:        vldtr,
		mt:           cpm,
		provMt:       provtelemetry.NewNoopMetrics(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

var _ (events.Registrar) = (*Server)(nil)

func (s *Server) initTracer() (*sdktrace.TracerProvider, error) {
	// create a stdout exporter to show collected spans out to stdout.
	exporter, err := stdout.New(stdout.WithPrettyPrint())
	if err != nil {
		return nil, err
	}
	sample_ratio := s.cfg.Tracing.SampleRatio
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sample_ratio))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp, nil
}

// Register implements events.Registrar
func (s *Server) Register(topic string, handler events.Handler, mdw ...message.HandlerMiddleware) {
	s.evt.Register(topic, handler, mdw...)
}

// ConsumeEvents implements events.Registrar
func (s *Server) ConsumeEvents(c ...events.Consumer) {
	s.evt.ConsumeEvents(c...)
}

func initMetrics(r sdkmetric.Reader) *sdkmetric.MeterProvider {
	// See the go.opentelemetry.io/otel/sdk/resource package for more
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("minder"),
		// TODO: Make this auto-generated
		semconv.ServiceVersion("v0.1.0"),
	)
	// By default/spec (?!), otel includes net.sock.peer.{addr,port}.
	// See https://github.com/open-telemetry/opentelemetry-go-contrib/issues/3071
	// This exposes a DoS vector and needlessly blows up the RPC metrics.
	// This view filters the peer address and port out of the metrics.
	rpcPeerFilter := sdkmetric.NewView(
		sdkmetric.Instrument{Scope: instrumentation.Scope{
			Name: "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc",
		}},
		sdkmetric.Stream{AttributeFilter: attribute.NewDenyKeysFilter(
			"net.sock.peer.addr", "net.sock.peer.port",
		)},
	)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(r),
		sdkmetric.WithView(rpcPeerFilter),
	)

	otel.SetMeterProvider(mp)

	return mp
}

// StartGRPCServer starts a gRPC server and blocks while serving.
func (s *Server) StartGRPCServer(ctx context.Context) error {
	lis, err := net.Listen("tcp", s.cfg.GRPCServer.GetAddress())
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// add logger and tracing (if enabled)
	interceptors := []grpc.UnaryServerInterceptor{
		// TODO: this has no test coverage!
		util.SanitizingInterceptor(),
		logger.Interceptor(),
		AuthUnaryInterceptor,
	}

	options := []grpc.ServerOption{
		grpc.Creds(insecure.NewCredentials()),
		grpc.ChainUnaryInterceptor(interceptors...),
	}

	otelGRPCOpts := s.getOTELGRPCInterceptorOpts()
	if len(otelGRPCOpts) > 0 {
		options = append(options, grpc.StatsHandler(otelgrpc.NewServerHandler()))
	}

	s.grpcServer = grpc.NewServer(
		options...,
	)

	// register the services (declared within register_handlers.go)
	RegisterGRPCServices(s)

	reflection.Register(s.grpcServer)

	errch := make(chan error)

	log.Printf("Starting gRPC server on %s", s.cfg.GRPCServer.GetAddress())

	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			errch <- fmt.Errorf("failed to serve: %w", err)
		}
	}()

	select {
	case err := <-errch:
		log.Printf("GRPC server fatal error: %v\n", err)
		return err
	case <-ctx.Done():
		log.Printf("shutting down 'GRPC server'")
		s.grpcServer.GracefulStop()
		return nil
	}
}

// getOTELGRPCInterceptorOpts gathers relevant options and
func (s *Server) getOTELGRPCInterceptorOpts() []otelgrpc.Option {
	opts := []otelgrpc.Option{}
	if s.cfg.Tracing.Enabled {
		opts = append(opts, otelgrpc.WithTracerProvider(otel.GetTracerProvider()))
	}

	if s.cfg.Metrics.Enabled {
		opts = append(opts, otelgrpc.WithMeterProvider(otel.GetMeterProvider()))
	}

	return opts
}

// StartHTTPServer starts a HTTP server and registers the gRPC handler mux to it
// set store as a blank identifier for now as we will use it in the future
func (s *Server) StartHTTPServer(ctx context.Context) error {

	mux := http.NewServeMux()

	addTracing := s.cfg.Tracing.Enabled

	if addTracing {
		tp, err := s.initTracer()
		if err != nil {
			return fmt.Errorf("failed to initialize TracerProvider: %w", err)
		}
		defer shutdownHandler("TracerProvider", func(ctx context.Context) error {
			return tp.Shutdown(ctx)
		})
	}

	gwmux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// register the services (declared within register_handlers.go)
	RegisterGatewayHTTPHandlers(ctx, gwmux, s.cfg.GRPCServer.GetAddress(), opts)

	fs := http.FileServer(http.FS(assets.StaticAssets))

	mw := otelhttp.NewMiddleware("webhook")

	mux.Handle("/", gwmux)
	mux.Handle("/api/v1/webhook/", mw(s.HandleGitHubWebHook()))
	mux.Handle("/static/", fs)

	errch := make(chan error)

	log.Printf("Starting HTTP server on %s", s.cfg.HTTPServer.GetAddress())

	server := http.Server{
		Addr:              s.cfg.HTTPServer.GetAddress(),
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	// start the metrics server if enabled
	if s.cfg.Metrics.Enabled {
		go func() {
			if err := s.startMetricServer(ctx); err != nil {
				log.Printf("failed to start metrics server: %v", err)
			}
		}()
	}

	// start the HTTP server
	go func() {
		if err := server.ListenAndServe(); err != nil {
			errch <- fmt.Errorf("failed to serve: %w", err)
		}
	}()

	select {
	case err := <-errch:
		log.Printf("HTTP server fatal error: %v", err)
		return err
	case <-ctx.Done():
		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownRelease()

		log.Printf("shutting down 'HTTP server'")

		return server.Shutdown(shutdownCtx)
	}
}

// startMetricServer starts a Prometheus metrics server and blocks while serving
func (s *Server) startMetricServer(ctx context.Context) error {
	// pull-based Prometheus exporter
	prometheusExporter, err := prometheus.New(
		prometheus.WithNamespace("minder"),
	)
	if err != nil {
		return fmt.Errorf("could not initialize metrics: %w", err)
	}

	mp := initMetrics(prometheusExporter)
	defer shutdownHandler("MeterProvider", func(ctx context.Context) error {
		return mp.Shutdown(ctx)
	})

	err = s.mt.initInstruments(s.store)
	if err != nil {
		return fmt.Errorf("could not initialize instruments: %w", err)
	}

	handler := promhttp.Handler()
	mux := http.NewServeMux()
	mux.Handle(metricsPath, handler)

	ch := make(chan error)

	log.Printf("Starting metrics server on %s", s.cfg.MetricServer.GetAddress())

	server := http.Server{
		Addr:              s.cfg.MetricServer.GetAddress(),
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	go func() {
		ch <- server.ListenAndServe()
	}()

	select {
	case err := <-ch:
		log.Printf("Metric server fatal error: %v", err)
		return err
	case <-ctx.Done():
		shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownRelease()

		log.Printf("shutting down 'Metric server'")

		return server.Shutdown(shutdownCtx)
	}
}

// HandleEvents starts the event handler and blocks while handling events.
func (s *Server) HandleEvents(ctx context.Context) func() error {
	return func() error {
		defer s.evt.Close()
		return s.evt.Run(ctx)
	}
}

type shutdowner func(context.Context) error

func shutdownHandler(component string, sdf shutdowner) {
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownRelease()

	log.Printf("shutting down '%s'", component)

	if err := sdf(shutdownCtx); err != nil {
		log.Fatalf("error shutting down '%s': %+v", component, err)
	}
}
