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
// It does make a good example of how to

package auth

import (
	"context"
	"encoding/json"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"log"
	"net"
	"os"
	"reflect"
	"testing"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/proto/v1"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterLogInServiceServer(s, &mockLogInServiceServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestSaveCredentials(t *testing.T) {
	// Arrange
	expectedCreds := Credentials{
		AccessToken:           "testAccessToken",
		RefreshToken:          "testRefreshToken",
		AccessTokenExpiresIn:  3600,
		RefreshTokenExpiresIn: 7200,
	}

	os.Setenv("XDG_CONFIG_HOME", "/tmp")

	// Act
	filePath, err := saveCredentials(expectedCreds)
	if err != nil {
		t.Fatalf("saveCredentials returned unexpected error: %v", err)
	}

	// Assert
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var actualCreds Credentials
	if err := json.Unmarshal(data, &actualCreds); err != nil {
		t.Fatalf("Failed to unmarshal credentials: %v", err)
	}

	if !reflect.DeepEqual(actualCreds, expectedCreds) {
		t.Fatalf("Expected %+v, got %+v", expectedCreds, actualCreds)
	}

	// Clean up ones mess
	os.Unsetenv("XDG_CONFIG_HOME")
	os.Remove(filePath)
}

func TestGetLoginServiceClient(t *testing.T) {

	s := grpc.NewServer()
	pb.RegisterLogInServiceServer(s, &mockLogInServiceServer{})
	lis = bufconn.Listen(bufSize)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	defer s.Stop()

	ctx := context.Background()
	creds, err := getLoginServiceClient(ctx, "bufnet", "testUser", "testPass", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("getLoginServiceClient returned unexpected error: %v", err)
	}

	if creds.AccessToken != "mockAccessToken" || creds.RefreshToken != "mockRefreshToken" {
		t.Fatalf("Expected mockAccessToken and mockRefreshToken, got %s and %s", creds.AccessToken, creds.RefreshToken)
	}
}

type mockLogInServiceServer struct {
	pb.UnimplementedLogInServiceServer
}

func (s *mockLogInServiceServer) LogIn(ctx context.Context, in *pb.LogInRequest) (*pb.LogInResponse, error) {
	return &pb.LogInResponse{
		AccessToken:           "mockAccessToken",
		RefreshToken:          "mockRefreshToken",
		AccessTokenExpiresIn:  3600,
		RefreshTokenExpiresIn: 7200,
	}, nil
}
