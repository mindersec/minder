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

// generateState generates a random string of length n, used as the OAuth state
func generateState(n int) (string, error) {
	randomBytes := make([]byte, n)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	state := base64.RawURLEncoding.EncodeToString(randomBytes)
	return state, nil
}

// CheckHealth is a simple health check for monitoring
func (s *Server) CheckHealth(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Status: "OK"}, nil
}

// newOAuthConfig creates a new OAuth2 config for the given provider
// and whether the client is a CLI or web client
func (s *Server) newOAuthConfig(provider string, cli bool) (*oauth2.Config, error) {
	redirectURL := func(provider string, cli bool) string {
		if cli {
			return fmt.Sprintf("http://localhost:8080/api/v1/auth/callback/%s/cli", provider)
		}
		return fmt.Sprintf("http://localhost:8080/api/v1/auth/callback/%s/web", provider)
	}

	scopes := func(provider string) []string {
		if provider == "google" {
			return []string{"profile", "email"}
		}
		return []string{"user:email"}
	}

	endpoint := func(provider string) oauth2.Endpoint {
		if provider == "google" {
			return google.Endpoint
		}
		return github.Endpoint
	}

	if provider != "google" && provider != "github" {
		return nil, fmt.Errorf("invalid provider: %s", provider)
	}

	return &oauth2.Config{
		ClientID:     viper.GetString(fmt.Sprintf("%s.client_id", provider)),
		ClientSecret: viper.GetString(fmt.Sprintf("%s.client_secret", provider)),
		RedirectURL:  redirectURL(provider, cli),
		Scopes:       scopes(provider),
		Endpoint:     endpoint(provider),
	}, nil
}

// GetAuthorizationURL returns the URL to redirect the user to for authorization
// and the state to be used for the callback. It accepts a provider string
// and a boolean indicating whether the client is a CLI or web client
func (s *Server) GetAuthorizationURL(ctx context.Context, req *pb.AuthorizationURLRequest) (*pb.AuthorizationURLResponse, error) {
	oauthConfig, err := s.newOAuthConfig(req.Provider, req.Cli)
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

// ExchangeCodeForTokenCLI exchanges an OAuth2 code for a token
// This is specific for CLI clients which require a different
func (s *Server) ExchangeCodeForTokenCLI(ctx context.Context, in *pb.CodeExchangeRequestCLI) (*pb.CodeExchangeResponseCLI, error) {
	oauthConfig, err := s.newOAuthConfig(in.Provider, true)
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
	var status string
	if token.Valid() {
		status = "success"
	} else {
		status = "failure"
	}

	cliAppURL := "http://localhost:8891/shutdown" // Replace PORT with the appropriate port number

	resp, err := http.Post(cliAppURL, "application/json", bytes.NewBuffer([]byte(`{"status": "`+status+`"}`)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to send status to CLI application, status code: %d", resp.StatusCode)
	}

	return &pb.CodeExchangeResponseCLI{
		Html: "You can now close this window.",
	}, nil
}

// ExchangeCodeForTokenWEB exchanges an OAuth2 code for a token and returns
// a JWT token as a session cookie. This handler is specific for web clients.
func (s *Server) ExchangeCodeForTokenWEB(ctx context.Context, in *pb.CodeExchangeRequestWEB) (*pb.CodeExchangeResponseWEB, error) {
	oauthConfig, err := s.newOAuthConfig(in.Provider, false)
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

	// TODO: The below response needs to return as a session cookie containing the JWT token
	// Once the JWT code is implemented.
	// http.SetCookie(w, &http.Cookie{
	// 	Name    "access_token",
	// 	Value   JWT token,
	// 	Expires time.Now().Add(24 * time.Hour),
	// })

	//
	return &pb.CodeExchangeResponseWEB{
		AccessToken: token.AccessToken,
	}, nil
}
