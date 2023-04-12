// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package services

import (
	"context"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/proto/v1"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type Server struct {
	pb.UnimplementedHealthServiceServer
	pb.UnimplementedAuthUrlServiceServer
	OAuth2       *oauth2.Config
	ClientID     string
	ClientSecret string
}

func (s *Server) CheckHealth(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{Status: "OK"}, nil
}

func NewServer(oauth2Config *oauth2.Config) *Server {
	return &Server{
		OAuth2: oauth2Config,
	}
}

func (s *Server) AuthUrl(ctx context.Context, req *pb.AuthUrlRequest) (*pb.AuthUrlResponse, error) {
	oauth2Cfg := &oauth2.Config{
		ClientID:     s.ClientID,
		ClientSecret: s.ClientSecret,
		Endpoint:     github.Endpoint,
		RedirectURL:  "http://localhost:8080/api/v1/callback",
		Scopes:       []string{"user:email"},
	}
	url := oauth2Cfg.AuthCodeURL("state")

	response := &pb.AuthUrlResponse{
		Url: url,
	}
	return response, nil
}
