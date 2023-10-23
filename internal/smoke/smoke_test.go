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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package smoke

import (
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

func grpcConnect(t *testing.T) *grpc.ClientConn {
	grpcHost := os.Getenv("GRPC_HOST")
	assert.NotEmpty(t, grpcHost, "GRPC_HOST environment variable is not set")

	conn, err := grpc.Dial(grpcHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	assert.NoError(t, err, "Failed to dial gRPC server")

	return conn
}

func TestGRPCHealthCheck(t *testing.T) {
	conn := grpcConnect(t)
	defer conn.Close()

	client := pb.NewHealthServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.CheckHealth(ctx, &pb.CheckHealthRequest{}, grpc.WaitForReady(true))
	assert.NoError(t, err, "Failed to check health")

	assert.Equal(t, "OK", resp.Status, "Expected health check status to be 'ok'")
}

func TestHTTPHealthCheck(t *testing.T) {
	httpHost := os.Getenv("HTTP_HOST")
	assert.NotEmpty(t, httpHost, "HTTP_HOST environment variable is not set")

	resp, err := http.Get(httpHost + "/api/v1/health")
	assert.NoError(t, err, "Failed to make HTTP request")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err, "Failed to read response body")

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected status 200 OK")
	assert.Equal(t, `{"status":"OK"}`, string(body), "Unexpected response body")
}

func TestAuthUserService(t *testing.T) {
	conn := grpcConnect(t)
	defer conn.Close()

	client := pb.NewUserServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.GetUser(ctx, &pb.GetUserRequest{})
	assert.Error(t, err, "Expected GetUser to fail without credentials")

	_, err = client.CreateUser(ctx, &pb.CreateUserRequest{})
	assert.Error(t, err, "Expected CreateUser to fail without credentials")

	_, err = client.DeleteUser(ctx, &pb.DeleteUserRequest{})
	assert.Error(t, err, "Expected DeleteUser to fail without credentials")
}

func TestGetAuthorizationURL(t *testing.T) {
	owner := "stacklok"
	ctx := context.Background()
	conn := grpcConnect(t)
	defer conn.Close()

	client := pb.NewOAuthServiceClient(conn)

	_, err := client.GetAuthorizationURL(ctx, &pb.GetAuthorizationURLRequest{
		Provider:  "github",
		ProjectId: "test",
		Cli:       true,
		Port:      8080,
		Owner:     &owner,
	})
	assert.Error(t, err, "Expected failure to get authorization URL")
}

func TestStoreProviderToken(t *testing.T) {
	owner := "stacklok"
	ctx := context.Background()
	conn := grpcConnect(t)
	defer conn.Close()

	client := pb.NewOAuthServiceClient(conn)

	_, err := client.StoreProviderToken(ctx, &pb.StoreProviderTokenRequest{
		Provider:    "github",
		ProjectId:   "test",
		AccessToken: "test",
		Owner:       &owner,
	})
	assert.Error(t, err, "Expected failure to store provider token")
}

func TestRevokeProviderToken(t *testing.T) {

	ctx := context.Background()
	conn := grpcConnect(t)
	defer conn.Close()

	client := pb.NewOAuthServiceClient(conn)

	_, err := client.RevokeOauthTokens(ctx, &pb.RevokeOauthTokensRequest{})
	assert.Error(t, err, "Expected failure to revoke oauth tokens")
}
