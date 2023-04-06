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

// Test coverage for pkg/services/services.go is currently handled by cmd/server/app/serve_test.go
// We could move if it makes sense at some point.

// import (
// 	"context"
// 	"net/http"
// 	"net/http/httptest"
// 	"strings"
// 	"testing"

// 	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/proto/v1"
// 	"golang.org/x/oauth2"
// )

// func TestCheckHealth(t *testing.T) {
// 	server := NewServer(nil)

// 	req := &pb.HealthRequest{}
// 	resp, err := server.CheckHealth(context.Background(), req)

// 	if err != nil {
// 		t.Errorf("Unexpected error: %v", err)
// 	}

// 	expectedStatus := "OK"
// 	if resp.Status != expectedStatus {
// 		t.Errorf("Expected status %q, got %q", expectedStatus, resp.Status)
// 	}
// }

// func TestAuthHTTPUrl(t *testing.T) {
// 	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(http.StatusTemporaryRedirect)
// 	}))
// 	defer ts.Close()

// 	oauth2Config := &oauth2.Config{
// 		RedirectURL: ts.URL,
// 	}

// 	server := NewServer(oauth2Config)
// 	req := &pb.AuthUrlRequest{}
// 	resp, err := server.AuthUrl(context.Background(), req)

// 	if err != nil {
// 		t.Errorf("Unexpected error: %v", err)
// 	}

// 	if !strings.HasPrefix(resp.Url, "https://github.com/login/oauth/authorize") {
// 		t.Errorf("Expected URL to start with 'https://github.com/login/oauth/authorize', got %q", resp.Url)
// 	}
// }
