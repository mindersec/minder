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
	"database/sql"
	"testing"

	mockdb "github.com/stacklok/mediator/database/mock"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"

	"github.com/golang/mock/gomock"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"google.golang.org/grpc/codes"
)

func TestCheckHealth(t *testing.T) {

	server := Server{}
	response, err := server.CheckHealth(context.Background(), &pb.CheckHealthRequest{})
	if err != nil {
		t.Errorf("Error in CheckHealth: %v", err)
	}

	if response.Status != "OK" {
		t.Errorf("Unexpected response from CheckHealth: %v", response)
	}
}

func TestNewOAuthConfig(t *testing.T) {

	// Test with CLI set
	config, err := newOAuthConfig("google", true)
	if err != nil {
		t.Errorf("Error in newOAuthConfig: %v", err)
	}

	if config.Endpoint != google.Endpoint {
		t.Errorf("Unexpected endpoint: %v", config.Endpoint)
	}

	// Test with CLI set
	config, err = newOAuthConfig("github", true)
	if err != nil {
		t.Errorf("Error in newOAuthConfig: %v", err)
	}

	if config.Endpoint != github.Endpoint {
		t.Errorf("Unexpected endpoint: %v", config.Endpoint)
	}

	// Test with CLI set
	config, err = newOAuthConfig("google", false)
	if err != nil {
		t.Errorf("Error in newOAuthConfig: %v", err)
	}

	if config.Endpoint != google.Endpoint {
		t.Errorf("Unexpected endpoint: %v", config.Endpoint)
	}

	// Test with CLI set
	config, err = newOAuthConfig("github", false)
	if err != nil {
		t.Errorf("Error in newOAuthConfig: %v", err)
	}

	if config.Endpoint != github.Endpoint {
		t.Errorf("Unexpected endpoint: %v", config.Endpoint)
	}

	_, err = newOAuthConfig("invalid", true)
	if err == nil {
		t.Errorf("Expected error in newOAuthConfig, but got nil")
	}
}

func TestGetAuthorizationURL(t *testing.T) {
	state := "test"
	grpID := sql.NullInt32{Int32: 1, Valid: true}
	port := sql.NullInt32{Int32: 8080, Valid: true}

	testCases := []struct {
		name               string
		req                *pb.GetAuthorizationURLRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetAuthorizationURLResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.GetAuthorizationURLRequest{
				Provider: "github",
				Port:     8080,
				Cli:      true,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateSessionState(gomock.Any(), gomock.Any()).
					Return(db.SessionStore{
						GrpID:        grpID,
						Port:         port,
						SessionState: state,
					}, nil)
				store.EXPECT().
					DeleteSessionStateByGroupID(gomock.Any(), gomock.Any()).
					Return(nil)
			},

			checkResponse: func(t *testing.T, res *pb.GetAuthorizationURLResponse, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				if res.Url == "" {
					t.Errorf("Unexpected response from GetAuthorizationURL: %v", res)
				}
			},

			expectedStatusCode: codes.OK,
		},
	}
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := Server{store: store}

			res, err := server.GetAuthorizationURL(context.Background(), tc.req)
			tc.checkResponse(t, res, err)
		})
	}
}
