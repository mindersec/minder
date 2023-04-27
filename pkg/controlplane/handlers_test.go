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

package controlplane

import (
	"context"
	"testing"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/proto/v1"

	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

func TestCheckHealth(t *testing.T) {
	server := Server{}

	response, err := server.CheckHealth(context.Background(), &pb.HealthRequest{})
	if err != nil {
		t.Errorf("Error in CheckHealth: %v", err)
	}

	if response.Status != "OK" {
		t.Errorf("Unexpected response from CheckHealth: %v", response)
	}
}

func TestGenerateState(t *testing.T) {
	state, err := generateState(32)
	if err != nil {
		t.Errorf("Error in generateState: %v", err)
	}

	if len(state) != 43 {
		t.Errorf("Unexpected length of state: %v", len(state))
	}
}

func TestNewOAuthConfig(t *testing.T) {
	server := Server{}

	config, err := server.newOAuthConfig("google", true)
	if err != nil {
		t.Errorf("Error in newOAuthConfig: %v", err)
	}

	if config.Endpoint != google.Endpoint {
		t.Errorf("Unexpected endpoint: %v", config.Endpoint)
	}

	config, err = server.newOAuthConfig("github", true)
	if err != nil {
		t.Errorf("Error in newOAuthConfig: %v", err)
	}

	if config.Endpoint != github.Endpoint {
		t.Errorf("Unexpected endpoint: %v", config.Endpoint)
	}

	_, err = server.newOAuthConfig("invalid", true)
	if err == nil {
		t.Errorf("Expected error in newOAuthConfig, but got nil")
	}
}

func TestGetAuthorizationURL(t *testing.T) {
	server := Server{}

	response, err := server.GetAuthorizationURL(context.Background(), &pb.AuthorizationURLRequest{Provider: "google"})
	if err != nil {
		t.Errorf("Error in GetAuthorizationURL: %v", err)
	}

	if response.Url == "" {
		t.Errorf("Unexpected response from GetAuthorizationURL: %v", response)
	}
}
