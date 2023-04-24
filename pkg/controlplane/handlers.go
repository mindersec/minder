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

package controlplane

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/proto/v1"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/spf13/viper"
)

type Server struct {
	pb.UnimplementedHealthServiceServer
	pb.UnimplementedOAuthServiceServer
	OAuth2       *oauth2.Config
	ClientID     string
	ClientSecret string
}

func generateState(n int) (string, error) {
	randomBytes := make([]byte, n)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	state := base64.RawURLEncoding.EncodeToString(randomBytes)
	return state, nil
}

func (s *Server) CheckHealth(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Status: "OK"}, nil
}

func (s *Server) newOAuthConfig(provider string) (*oauth2.Config, error) {
	// provider = "github"
	switch provider {
	case "google":
		return &oauth2.Config{
			ClientID:     viper.GetString("google.client_id"),
			ClientSecret: viper.GetString("google.client_secret"),
			RedirectURL:  "http://localhost:8080/auth/google/callback",
			Scopes:       []string{"profile", "email"},
			Endpoint:     google.Endpoint,
		}, nil
	case "github":
		return &oauth2.Config{
			ClientID:     viper.GetString("github.client_id"),
			ClientSecret: viper.GetString("github.client_secret"),
			RedirectURL:  "http://localhost:8080/api/v1/auth/callback/github",
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		}, nil
	default:
		return nil, fmt.Errorf("invalid provider: %s", provider)
	}
}

func (s *Server) GetAuthorizationURL(ctx context.Context, req *pb.AuthorizationURLRequest) (*pb.AuthorizationURLResponse, error) {
	oauthConfig, err := s.newOAuthConfig(req.Provider)
	if err != nil {
		return nil, err
	}
	state, err := generateState(32)

	if err != nil {
		fmt.Println("Error generating state:", err)
		return nil, err
	}
	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)

	response := &pb.AuthorizationURLResponse{
		Url: url,
	}
	return response, nil
}

func (s *Server) ExchangeCodeForToken(ctx context.Context, in *pb.CodeExchangeRequest) (*pb.CodeExchangeResponse, error) {

	oauthConfig, err := s.newOAuthConfig(in.Provider)
	if err != nil {
		return nil, err
	}

	if oauthConfig == nil {
		return nil, fmt.Errorf("oauth2.Config is nil")
	}

	token, err := oauthConfig.Exchange(ctx, in.Code)
	if err != nil {
		return nil, err
	}

	// check if the token is valid
	var requestBody []byte
	if token.Valid() {
		requestBody = []byte(`{"status": "success"}`)
	} else {
		requestBody = []byte(`{"status": "failure"}`)
	}

	cliAppURL := "http://localhost:8891/shutdown" // Replace PORT with the appropriate port number

	resp, err := http.Post(cliAppURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to send status to CLI application, status code: %d", resp.StatusCode)
	}

	return &pb.CodeExchangeResponse{
		AccessToken: token.AccessToken,
		Status:      "success",
	}, nil
}
