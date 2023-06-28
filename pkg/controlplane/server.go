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
	"log"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
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

	_ "github.com/lib/pq" // nolint

	"github.com/stacklok/mediator/internal/logger"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

const metricsPath = "/metrics"

// Server represents the controlplane server
type Server struct {
	store      db.Store
	grpcServer *grpc.Server
	pb.UnimplementedHealthServiceServer
	pb.UnimplementedOAuthServiceServer
	pb.UnimplementedAuthServiceServer
	pb.UnimplementedOrganizationServiceServer
	pb.UnimplementedGroupServiceServer
	pb.UnimplementedRoleServiceServer
	pb.UnimplementedUserServiceServer
	pb.UnimplementedRepositoryServiceServer
	OAuth2       *oauth2.Config
	ClientID     string
	ClientSecret string
}

// NewServer creates a new server instance
func NewServer(store db.Store) *Server {
	server := &Server{
		store: store,
	}
	return server
}

func initTracer() (*sdktrace.TracerProvider, error) {
	// create a stdout exporter to show collected spans out to stdout.
	exporter, err := stdout.New(stdout.WithPrettyPrint())
	if err != nil {
		return nil, err
	}
	// for the demonstration, we use AlwaysSmaple sampler to take all spans.
	// do not use this option in production.
	viper.SetDefault("tracing.sample_ratio", 0.1)
	sample_ratio := viper.GetFloat64("tracing.sample_ratio")
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sample_ratio))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp, nil
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
func (s *Server) StartGRPCServer(address string, store db.Store) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	server := NewServer(store)

	if err != nil {
		log.Fatal("Cannot create server: ", err)
	}

	log.Println("Initializing logger in level: " + viper.GetString("logging.level"))

	viper.SetDefault("logging.level", "debug")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.logFile", "")
	viper.SetDefault("tracing.enabled", false)

	// add logger and tracing (if enabled)
	interceptors := []grpc.UnaryServerInterceptor{}
	interceptors = append(interceptors, logger.Interceptor(viper.GetString("logging.level"),
		viper.GetString("logging.format"), viper.GetString("logging.logFile")))
	interceptors = append(interceptors, AuthUnaryInterceptor)

	addTracing := viper.GetBool("tracing.enabled")
	addMetrics := viper.GetBool("metrics.enabled")

	otelGRPCOpts := getOTELGRPCInterceptorOpts(addTracing, addMetrics)
	if len(otelGRPCOpts) > 0 {
		interceptors = append(interceptors, otelgrpc.UnaryServerInterceptor(otelGRPCOpts...))
	}

	s.grpcServer = grpc.NewServer(
		grpc.Creds(insecure.NewCredentials()),
		grpc.ChainUnaryInterceptor(interceptors...),
	)

	server.grpcServer = s.grpcServer

	// register the services (declared within register_handlers.go)
	RegisterGRPCServices(server)

	reflection.Register(s.grpcServer)

	log.Printf("Starting gRPC server on %s", address)
	if err := s.grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// getOTELGRPCInterceptorOpts gathers relevant options and
func getOTELGRPCInterceptorOpts(addTracing, addMetrics bool) []otelgrpc.Option {
	opts := []otelgrpc.Option{}
	if addTracing {
		opts = append(opts, otelgrpc.WithTracerProvider(otel.GetTracerProvider()))
	}

	if addMetrics {
		opts = append(opts, otelgrpc.WithMeterProvider(otel.GetMeterProvider()))
	}

	return opts
}

// StartHTTPServer starts a HTTP server and registers the gRPC handler mux to it
// set store as a blank identifier for now as we will use it in the future
func StartHTTPServer(address, grpcAddress string, store db.Store) {

	mux := http.NewServeMux()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	viper.SetDefault("tracing.enabled", false)
	addTracing := viper.GetBool("tracing.enabled")

	if addTracing {
		tp, err := initTracer()
		if err != nil {
			log.Fatalf("failed to initialize TracerProvider: %v", err)
		}
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Fatalf("error shutting down TracerProvider: %v", err)
			}
		}()
	}

	viper.SetDefault("metrics.enabled", false)
	addMeter := viper.GetBool("metrics.enabled")

	if addMeter {
		// Pull-based Prometheus exporter
		prometheusExporter, err := prometheus.New(
			prometheus.WithNamespace("mediator"),
		)
		if err != nil {
			log.Fatalf("could not initialize metrics: %v", err)
		}

		defer func() {
			if err := prometheusExporter.Shutdown(ctx); err != nil {
				log.Fatalf("error shutting down PrometheusExporter: %v", err)
			}
		}()

		mp := initMetrics(prometheusExporter)
		defer func() {
			if err := mp.Shutdown(ctx); err != nil {
				log.Fatalf("error shutting down MeterProvider: %v", err)
			}
		}()

		handler := promhttp.Handler()
		mux.Handle(metricsPath, handler)
	}

	gwmux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// register the services (declared within register_handlers.go)
	RegisterGatewayHTTPHandlers(ctx, gwmux, grpcAddress, opts)

	mux.Handle("/", gwmux)
	mux.HandleFunc("/api/v1/webhook/", HandleGitHubWebHook(store))

	log.Printf("Starting HTTP server on %s", address)
	if err := http.ListenAndServe(address, mux); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
