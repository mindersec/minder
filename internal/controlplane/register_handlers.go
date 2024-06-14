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

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
}
