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

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// RegisterGatewayHTTPHandlers registers the gateway HTTP handlers
func RegisterGatewayHTTPHandlers(ctx context.Context, gwmux *runtime.ServeMux, grpcAddress string, opts []grpc.DialOption) {
	// Register HealthService handler
	if err := pb.RegisterHealthServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatalf("failed to register gateway: %v", err)
	}

	// Register CallBackService handler
	if err := pb.RegisterOAuthServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatalf("failed to register gateway: %v", err)
	}

	// Register LoginService handler
	if err := pb.RegisterLogInServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatalf("failed to register gateway: %v", err)
	}

	// Register OrganizationService handler
	if err := pb.RegisterOrganizationServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatalf("failed to register gateway: %v", err)
	}

	// Register GroupService handler
	if err := pb.RegisterGroupServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatalf("failed to register gateway: %v", err)
	}

	// Register RoleService handler
	if err := pb.RegisterRoleServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatalf("failed to register gateway: %v", err)
	}

	// Register UserService handler
	if err := pb.RegisterUserServiceHandlerFromEndpoint(ctx, gwmux, grpcAddress, opts); err != nil {
		log.Fatalf("failed to register gateway: %v", err)
	}

}

// RegisterGRPCServices registers the GRPC services
func RegisterGRPCServices(s *Server) {
	// Register HealthService handler
	pb.RegisterHealthServiceServer(s.grpcServer, s)

	// Register AuthUrlService handler
	pb.RegisterOAuthServiceServer(s.grpcServer, s)

	// Register the Login service
	pb.RegisterLogInServiceServer(s.grpcServer, s)

	// Register the Logout service
	pb.RegisterLogOutServiceServer(s.grpcServer, s)

	// Register the Organization service
	pb.RegisterOrganizationServiceServer(s.grpcServer, s)

	// Register the Groups service
	pb.RegisterGroupServiceServer(s.grpcServer, s)
	// Register the Role service
	pb.RegisterRoleServiceServer(s.grpcServer, s)

	// Register the User service
	pb.RegisterUserServiceServer(s.grpcServer, s)

}
