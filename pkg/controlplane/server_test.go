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
	"testing"
	"time"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	// gRPC server
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	// RegisterAuthUrlServiceServer
	pb.RegisterHealthServiceServer(s, &Server{}) //
	pb.RegisterOAuthServiceServer(s, &Server{
		ClientID:     "test",
		ClientSecret: "test",
	})
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
	// HTTP server
	mux := http.NewServeMux()

	srv := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func getgRPCConnection() (*grpc.ClientConn, error) {
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func TestHealth(t *testing.T) {
	conn, err := getgRPCConnection()
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewHealthServiceClient(conn)
	_, err = client.CheckHealth(context.Background(), &pb.CheckHealthRequest{})
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
}

func TestWebhook(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/api/v1/github/hook")
	if err != nil {
		t.Fatalf("Failed to get webhook: %v", err)
	}
	defer resp.Body.Close()
}
