// Copyright 2023 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"google.golang.org/grpc/codes"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/db"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestNewOAuthConfig(t *testing.T) {
	t.Parallel()

	// Test with CLI set
	cfg, err := auth.NewOAuthConfig("google", true)
	if err != nil {
		t.Errorf("Error in newOAuthConfig: %v", err)
	}

	if cfg.Endpoint != google.Endpoint {
		t.Errorf("Unexpected endpoint: %v", cfg.Endpoint)
	}

	// Test with CLI set
	cfg, err = auth.NewOAuthConfig("github", true)
	if err != nil {
		t.Errorf("Error in newOAuthConfig: %v", err)
	}

	if cfg.Endpoint != github.Endpoint {
		t.Errorf("Unexpected endpoint: %v", cfg.Endpoint)
	}

	// Test with CLI set
	cfg, err = auth.NewOAuthConfig("google", false)
	if err != nil {
		t.Errorf("Error in newOAuthConfig: %v", err)
	}

	if cfg.Endpoint != google.Endpoint {
		t.Errorf("Unexpected endpoint: %v", cfg.Endpoint)
	}

	// Test with CLI set
	cfg, err = auth.NewOAuthConfig("github", false)
	if err != nil {
		t.Errorf("Error in newOAuthConfig: %v", err)
	}

	if cfg.Endpoint != github.Endpoint {
		t.Errorf("Unexpected endpoint: %v", cfg.Endpoint)
	}

	_, err = auth.NewOAuthConfig("invalid", true)
	if err == nil {
		t.Errorf("Expected error in newOAuthConfig, but got nil")
	}
}

func TestGetAuthorizationURL(t *testing.T) {
	t.Parallel()

	state := "test"
	orgID := uuid.New()
	projectID := uuid.New()
	port := sql.NullInt32{Int32: 8080, Valid: true}
	providerID := uuid.New()

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
					ListProvidersByProjectID(gomock.Any(), projectID).
					Return([]db.Provider{{
						ID:   providerID,
						Name: "github",
					}}, nil)
				store.EXPECT().
					CreateSessionState(gomock.Any(), gomock.Any()).
					Return(db.SessionStore{
						ProjectID:    projectID,
						Port:         port,
						SessionState: state,
					}, nil)
				store.EXPECT().
					DeleteSessionStateByProjectID(gomock.Any(), gomock.Any()).
					Return(nil)
			},

			checkResponse: func(t *testing.T, res *pb.GetAuthorizationURLResponse, err error) {
				t.Helper()

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

	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projectID},
		IsStaff:        true, // TODO: remove this
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projectID, OrganizationID: orgID}},
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newDefaultServer(t, store)

			res, err := server.GetAuthorizationURL(ctx, tc.req)
			tc.checkResponse(t, res, err)
		})
	}
}
