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

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	_ "github.com/lib/pq" // This is required for the postgres driver
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/internal/events"
	"github.com/stacklok/mediator/internal/logger"
	"github.com/stacklok/mediator/internal/util"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
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
	grpcServer *grpc.Server
	pb.UnimplementedHealthServiceServer
	pb.UnimplementedOAuthServiceServer
	pb.UnimplementedAuthServiceServer
	pb.UnimplementedOrganizationServiceServer
	pb.UnimplementedGroupServiceServer
	pb.UnimplementedRoleServiceServer
	pb.UnimplementedUserServiceServer
	pb.UnimplementedRepositoryServiceServer
	pb.UnimplementedPolicyServiceServer
	pb.UnimplementedKeyServiceServer
	OAuth2       *oauth2.Config
	ClientID     string
	ClientSecret string
}

// NewServer creates a new server instance
func NewServer(store db.Store, cfg *config.Config) (*Server, error) {
	// TODO: enable registering handlers with the router (arg 0)
	evt, err := events.Setup()
	if err != nil {
		log.Printf("Failed to set up eventer: %v", err)
		return nil, err
	}

	return &Server{
		store: store,
		cfg:   cfg,
		evt:   evt,
	}, nil
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
func (s *Server) Register(topic string, handler events.Handler) {
	s.evt.Register(topic, handler)
}

// ConsumeEvents implements events.Registrar
func (s *Server) ConsumeEvents(c ...events.Consumer) {
	s.evt.ConsumeEvents(c...)
}

func initMetrics(r sdkmetric.Reader) *sdkmetric.MeterProvider {
	// See the go.opentelemetry.io/otel/sdk/resource package for more
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("mediator"),
		// TODO: Make this auto-generated
		semconv.ServiceVersion("v0.1.0"),
	)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(r),
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
	otelGRPCOpts := s.getOTELGRPCInterceptorOpts()
	if len(otelGRPCOpts) > 0 {
		interceptors = append(interceptors, otelgrpc.UnaryServerInterceptor(otelGRPCOpts...))
	}

	s.grpcServer = grpc.NewServer(
		grpc.Creds(insecure.NewCredentials()),
		grpc.ChainUnaryInterceptor(interceptors...),
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

	addMeter := s.cfg.Metrics.Enabled

	if addMeter {
		// Pull-based Prometheus exporter
		prometheusExporter, err := prometheus.New(
			prometheus.WithNamespace("mediator"),
		)
		if err != nil {
			return fmt.Errorf("could not initialize metrics: %w", err)
		}

		mp := initMetrics(prometheusExporter)
		defer shutdownHandler("MeterProvider", func(ctx context.Context) error {
			return mp.Shutdown(ctx)
		})

		handler := promhttp.Handler()
		mux.Handle(metricsPath, handler)
	}

	gwmux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// register the services (declared within register_handlers.go)
	RegisterGatewayHTTPHandlers(ctx, gwmux, s.cfg.GRPCServer.GetAddress(), opts)

	mux.Handle("/", gwmux)
	mux.HandleFunc("/api/v1/webhook/", HandleGitHubWebHook(s.evt))

	errch := make(chan error)

	log.Printf("Starting HTTP server on %s", s.cfg.HTTPServer.GetAddress())

	server := http.Server{
		Addr:              s.cfg.HTTPServer.GetAddress(),
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}

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
