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
	"database/sql"
	"log"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/spf13/viper"
	"golang.org/x/oauth2"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"

	_ "github.com/lib/pq" // nolint

	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

type Server struct {
	store      db.Store
	grpcServer *grpc.Server
	pb.UnimplementedHealthServiceServer
	pb.UnimplementedOAuthServiceServer
	pb.UnimplementedLogInServiceServer
	OAuth2       *oauth2.Config
	ClientID     string
	ClientSecret string
}

func NewServer(store db.Store) *Server {
	server := &Server{
		store: store,
	}
	return server
}

func loggingInterceptor(level string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			for key, values := range md {
				for _, value := range values {
					log.Printf("[%s] header received: %s=%s", level, key, value)
				}
			}
		}
		resp, err := handler(ctx, req)
		log.Printf("[%s] method called: %s", level, info.FullMethod)
		log.Printf("[%s] incoming request: %v", level, req)

		log.Printf("[%s] outgoing response: %v", level, resp)
		return resp, err
	}
}

func (s *Server) StartGRPCServer(address string, dbConn string) {
	// store := db.NewStore(dbConn)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	conn, err := sql.Open("postgres", dbConn)
	if err != nil {
		log.Fatal("Cannot connect to DB: ", err)
	} else {
		log.Println("Connected to DB")
	}

	store := db.NewStore(conn)

	server := NewServer(store)

	if err != nil {
		log.Fatal("Cannot create server: ", err)
	}

	log.Println("Initializing logger in level: " + viper.GetString("logging.level"))

	if viper.GetString("logging.level") == "debug" {
		s.grpcServer = grpc.NewServer(
			grpc.Creds(insecure.NewCredentials()),
			grpc.UnaryInterceptor(loggingInterceptor(viper.GetString("logging.level"))),
		)
	} else {
		s.grpcServer = grpc.NewServer(
			grpc.Creds(insecure.NewCredentials()),
		)
	}

	server.grpcServer = s.grpcServer

	// register the services (declared within register_handlers.go)
	RegisterGRPCServices(server)

	reflection.Register(s.grpcServer)

	log.Printf("Starting gRPC server on %s", address)
	if err := s.grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func StartHTTPServer(address, grpcAddress string) {

	mux := http.NewServeMux()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	gwmux := runtime.NewServeMux()

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// register the services (declared within register_handlers.go)
	RegisterGatewayHTTPHandlers(ctx, gwmux, grpcAddress, opts)

	mux.Handle("/", gwmux)

	log.Printf("Starting HTTP server on %s", address)
	if err := http.ListenAndServe(address, mux); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
