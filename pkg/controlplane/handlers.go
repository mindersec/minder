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
	"context"
	"fmt"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/proto/v1"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/spf13/viper"
)

type Server struct {
	pb.UnimplementedHealthServiceServer
	pb.UnimplementedAuthUrlServiceServer
	ClientID     string
	ClientSecret string
}

func (s *Server) CheckHealth(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Status: "OK"}, nil
}

func (s *Server) newOAuthConfig(provider string) (*oauth2.Config, error) {
	fmt.Println("provider: ", provider)
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
			RedirectURL:  "http://localhost:8080/auth/github/callback",
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		}, nil
	default:
		return nil, fmt.Errorf("invalid provider: %s", provider)
	}
}

func (s *Server) AuthUrl(ctx context.Context, req *pb.AuthUrlRequest) (*pb.AuthUrlResponse, error) {
	oauthConfig, err := s.newOAuthConfig(req.Provider)
	if err != nil {
		return nil, err
	}
	url := oauthConfig.AuthCodeURL("state")

	response := &pb.AuthUrlResponse{
		Url: url,
	}
	return response, nil
}
