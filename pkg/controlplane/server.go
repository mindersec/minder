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

	"github.com/spf13/viper"
	"golang.org/x/oauth2"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"

	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	_ "github.com/lib/pq" // nolint

	"github.com/stacklok/mediator/internal/logger"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// Server represents the controlplane server
type Server struct {
	store      db.Store
	grpcServer *grpc.Server
	pb.UnimplementedHealthServiceServer
	pb.UnimplementedOAuthServiceServer
	pb.UnimplementedLogInServiceServer
	pb.UnimplementedLogOutServiceServer
	pb.UnimplementedOrganizationServiceServer
	pb.UnimplementedGroupServiceServer
	pb.UnimplementedRoleServiceServer
	pb.UnimplementedUserServiceServer
	pb.UnimplementedRevokeTokensServiceServer
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
	if addTracing {
		interceptorOpt := otelgrpc.WithTracerProvider(otel.GetTracerProvider())
		interceptors = append(interceptors, otelgrpc.UnaryServerInterceptor(interceptorOpt))
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
