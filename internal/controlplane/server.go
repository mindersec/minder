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
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gorilla/handlers"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/open-feature/go-sdk/openfeature"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/assets"
	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/auth/jwt"
	"github.com/stacklok/minder/internal/authz"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/controlplane/metrics"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/history"
	"github.com/stacklok/minder/internal/invites"
	"github.com/stacklok/minder/internal/logger"
	"github.com/stacklok/minder/internal/profiles"
	"github.com/stacklok/minder/internal/projects"
	"github.com/stacklok/minder/internal/providers"
	ghprov "github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/providers/github/service"
	"github.com/stacklok/minder/internal/providers/manager"
	"github.com/stacklok/minder/internal/providers/session"
	"github.com/stacklok/minder/internal/repositories/github"
	"github.com/stacklok/minder/internal/roles"
	"github.com/stacklok/minder/internal/ruletypes"
	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const metricsPath = "/metrics"

var (
	readHeaderTimeout = 2 * time.Second

	// RequestBodyMaxBytes is the maximum number of bytes that can be read from a request body
	// We limit to 2MB for now
	RequestBodyMaxBytes int64 = 2 << 20
)

// Server represents the controlplane server
type Server struct {
	store        db.Store
	cfg          *serverconfig.Config
	evt          events.Publisher
	mt           metrics.Metrics
	grpcServer   *grpc.Server
	jwt          jwt.Validator
	authzClient  authz.Client
	idClient     auth.Resolver
	cryptoEngine crypto.Engine
	featureFlags openfeature.IClient
	// We may want to start breaking up the server struct if we use it to
	// inject more entity-specific interfaces. For example, we may want to
	// consider having a struct per grpc service
	invites             invites.InviteService
	ruleTypes           ruletypes.RuleTypeService
	repos               github.RepositoryService
	roles               roles.RoleService
	profiles            profiles.ProfileService
	history             history.EvaluationHistoryService
	ghProviders         service.GitHubProviderService
	providerStore       providers.ProviderStore
	ghClient            ghprov.ClientService
	providerManager     manager.ProviderManager
	sessionService      session.ProviderSessionService
	providerAuthManager manager.AuthManager
	projectCreator      projects.ProjectCreator
	projectDeleter      projects.ProjectDeleter

	// Implementations for service registration
	pb.UnimplementedHealthServiceServer
	pb.UnimplementedOAuthServiceServer
	pb.UnimplementedUserServiceServer
	pb.UnimplementedRepositoryServiceServer
	pb.UnimplementedProjectsServiceServer
	pb.UnimplementedProfileServiceServer
	pb.UnimplementedRuleTypeServiceServer
	pb.UnimplementedArtifactServiceServer
	pb.UnimplementedPermissionsServiceServer
	pb.UnimplementedProvidersServiceServer
	pb.UnimplementedEvalResultsServiceServer
	pb.UnimplementedInviteServiceServer
}

// NewServer creates a new server instance
func NewServer(
	store db.Store,
	evt events.Publisher,
	cfg *serverconfig.Config,
	serverMetrics metrics.Metrics,
	jwtValidator jwt.Validator,
	cryptoEngine crypto.Engine,
	authzClient authz.Client,
	idClient auth.Resolver,
	inviteService invites.InviteService,
	repoService github.RepositoryService,
	roleService roles.RoleService,
	profileService profiles.ProfileService,
	historyService history.EvaluationHistoryService,
	ruleService ruletypes.RuleTypeService,
	ghProviders service.GitHubProviderService,
	providerManager manager.ProviderManager,
	providerAuthManager manager.AuthManager,
	providerStore providers.ProviderStore,
	sessionService session.ProviderSessionService,
	projectDeleter projects.ProjectDeleter,
	projectCreator projects.ProjectCreator,
	featureFlagClient *openfeature.Client,
) *Server {
	return &Server{
		store:               store,
		cfg:                 cfg,
		evt:                 evt,
		cryptoEngine:        cryptoEngine,
		jwt:                 jwtValidator,
		mt:                  serverMetrics,
		profiles:            profileService,
		history:             historyService,
		ruleTypes:           ruleService,
		providerStore:       providerStore,
		featureFlags:        featureFlagClient,
		ghClient:            &ghprov.ClientServiceImplementation{},
		providerManager:     providerManager,
		providerAuthManager: providerAuthManager,
		sessionService:      sessionService,
		invites:             inviteService,
		repos:               repoService,
		roles:               roleService,
		ghProviders:         ghProviders,
		authzClient:         authzClient,
		idClient:            idClient,
		projectCreator:      projectCreator,
		projectDeleter:      projectDeleter,
	}
}

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
		return fmt.Errorf("failed to listen: %v", err)
	}

	// add logger and tracing (if enabled)
	interceptors := []grpc.UnaryServerInterceptor{
		// TODO: this has no test coverage!
		util.SanitizingInterceptor(),
		logger.Interceptor(s.cfg.LoggingConfig),
		TokenValidationInterceptor,
		EntityContextProjectInterceptor,
		ProjectAuthorizationInterceptor,
		recovery.UnaryServerInterceptor(recovery.WithRecoveryHandlerContext(recoveryHandler)),
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

	otelmw := otelhttp.NewMiddleware("webhook")

	// Explicitly handle HTTP only requests
	err := gwmux.HandlePath(http.MethodGet, "/api/v1/auth/callback/{provider}/cli", s.HandleOAuthCallback())
	if err != nil {
		return fmt.Errorf("failed to register provider callback handler: %w", err)
	}
	err = gwmux.HandlePath(http.MethodGet, "/api/v1/auth/callback/{provider}/web", s.HandleOAuthCallback())
	if err != nil {
		return fmt.Errorf("failed to register provider callback handler: %w", err)
	}
	err = gwmux.HandlePath(http.MethodGet, "/api/v1/auth/callback/{provider}/app", s.HandleGitHubAppCallback())
	if err != nil {
		return fmt.Errorf("failed to register GitHub App callback handler: %w", err)
	}

	// This already has _some_ middleware due to the GRPC handling
	mux.Handle("/", withMaxSizeMiddleware(s.withCORSMiddleware(gwmux)))

	// This requires explicit middleware. CORS is not required here.
	mux.Handle("/api/v1/webhook/", otelmw(withMiddleware(s.HandleGitHubWebHook())))
	mux.Handle("/api/v1/ghapp/", otelmw(withMiddleware(s.HandleGitHubAppWebhook())))
	mux.Handle("/api/v1/gh-marketplace/", otelmw(withMiddleware(s.NoopWebhookHandler())))
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

func (s *Server) withCORSMiddleware(h http.Handler) http.Handler {
	if s.cfg.HTTPServer.CORS.Enabled {
		var opts []handlers.CORSOption
		if len(s.cfg.HTTPServer.CORS.AllowOrigins) > 0 {
			opts = append(opts, handlers.AllowedOrigins(s.cfg.HTTPServer.CORS.AllowOrigins))
		}
		if len(s.cfg.HTTPServer.CORS.AllowMethods) > 0 {
			opts = append(opts, handlers.AllowedMethods(s.cfg.HTTPServer.CORS.AllowMethods))
		}
		if len(s.cfg.HTTPServer.CORS.AllowHeaders) > 0 {
			opts = append(opts, handlers.AllowedHeaders(s.cfg.HTTPServer.CORS.AllowHeaders))
		}
		if len(s.cfg.HTTPServer.CORS.ExposeHeaders) > 0 {
			opts = append(opts, handlers.ExposedHeaders(s.cfg.HTTPServer.CORS.ExposeHeaders))
		}
		if s.cfg.HTTPServer.CORS.AllowCredentials {
			opts = append(opts, handlers.AllowCredentials())
		}

		return handlers.CORS(opts...)(h)
	}

	return h
}

// Sets up common middleawre
func withMiddleware(h http.Handler) http.Handler {
	l := log.Logger
	return handlers.RecoveryHandler(handlers.RecoveryLogger(&l))(withMaxSizeMiddleware(h))
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

	err = s.mt.Init(s.store)
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

func recoveryHandler(ctx context.Context, p any) error {
	log.Ctx(ctx).Error().
		Str("msg", "recovered from panic").
		Any("panic", p).
		Bytes("stack", debug.Stack())
	return status.Errorf(codes.Internal, "%s", p)
}

type shutdowner func(context.Context) error

func shutdownHandler(component string, sdf shutdowner) {
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownRelease()

	log.Printf("shutting down '%s'", component)

	if err := sdf(shutdownCtx); err != nil {
		log.Fatal().Msgf("error shutting down '%s': %+v", component, err)
	}
}

func withMaxSizeMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, RequestBodyMaxBytes)
		h.ServeHTTP(w, r)
	})
}
