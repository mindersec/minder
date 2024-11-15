// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	pb "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// RegisterGatewayHTTPHandlers registers the gateway HTTP handlers
func RegisterGatewayHTTPHandlers(ctx context.Context, gwmux *runtime.ServeMux, grpcAddress string, opts []grpc.DialOption) {
	// Register HealthService handler
	if err := pb.RegisterHealthServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatal().Err(err).Msg("failed to register gateway")
	}

	// Register CallBackService handler
	if err := pb.RegisterOAuthServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatal().Err(err).Msg("failed to register gateway")
	}

	// Register UserService handler
	if err := pb.RegisterUserServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatal().Err(err).Msg("failed to register gateway")
	}

	// Register the Repository service
	if err := pb.RegisterRepositoryServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatal().Err(err).Msg("failed to register gateway")
	}

	// Register the Profile service
	if err := pb.RegisterProfileServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatal().Err(err).Msg("failed to register gateway")
	}

	// Register the RuleType service
	if err := pb.RegisterRuleTypeServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatal().Err(err).Msg("failed to register gateway")
	}

	// Register the Package service
	if err := pb.RegisterArtifactServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatal().Err(err).Msg("failed to register gateway")
	}

	// Register the Permissions service
	if err := pb.RegisterPermissionsServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatal().Err(err).Msg("failed to register gateway")
	}

	// Register Providers service
	if err := pb.RegisterProvidersServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatal().Err(err).Msg("failed to register gateway")
	}

	// Register Projects service
	if err := pb.RegisterProjectsServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatal().Err(err).Msg("failed to register gateway")
	}

	// Register EvalResultsService handler
	if err := pb.RegisterEvalResultsServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatal().Err(err).Msg("failed to register gateway")
	}

	// Register the InviteService service
	if err := pb.RegisterInviteServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatal().Err(err).Msg("failed to register gateway")
	}

	// Register the DataSource service
	if err := pb.RegisterDataSourceServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatal().Err(err).Msg("failed to register gateway")
	}
}

// RegisterGRPCServices registers the GRPC services
func RegisterGRPCServices(s *Server) {
	// Register HealthService handler
	pb.RegisterHealthServiceServer(s.grpcServer, s)

	// Register AuthUrlService handler
	pb.RegisterOAuthServiceServer(s.grpcServer, s)

	// Register the User service
	pb.RegisterUserServiceServer(s.grpcServer, s)

	// Register the Repository service
	pb.RegisterRepositoryServiceServer(s.grpcServer, s)

	// Register the Profile service
	pb.RegisterProfileServiceServer(s.grpcServer, s)

	// Register the RuleType service
	pb.RegisterRuleTypeServiceServer(s.grpcServer, s)

	// Register the Artifact service
	pb.RegisterArtifactServiceServer(s.grpcServer, s)

	// Register the Permissions service
	pb.RegisterPermissionsServiceServer(s.grpcServer, s)

	// Register the Providers service
	pb.RegisterProvidersServiceServer(s.grpcServer, s)

	// Register the Projects service
	pb.RegisterProjectsServiceServer(s.grpcServer, s)

	// Register the EvalResultsService service
	pb.RegisterEvalResultsServiceServer(s.grpcServer, s)

	// Register the InviteService service
	pb.RegisterInviteServiceServer(s.grpcServer, s)

	// Register the DataSource service
	pb.RegisterDataSourceServiceServer(s.grpcServer, s)
}
