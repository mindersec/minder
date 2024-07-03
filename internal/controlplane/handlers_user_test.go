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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/auth/jwt"
	mockjwt "github.com/stacklok/minder/internal/auth/jwt/mock"
	"github.com/stacklok/minder/internal/authz/mock"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	mockcrypto "github.com/stacklok/minder/internal/crypto/mock"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/events"
	"github.com/stacklok/minder/internal/marketplaces"
	"github.com/stacklok/minder/internal/projects"
	"github.com/stacklok/minder/internal/providers"
	mockprov "github.com/stacklok/minder/internal/providers/github/service/mock"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	//nolint:gosec // not credentials, just an endpoint
	tokenEndpoint = "/realms/stacklok/protocol/openid-connect/token"
)

func TestCreateUser_gRPC(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	keyCloakUserToken := openid.New()
	require.NoError(t, keyCloakUserToken.Set("gh_id", "31337"))

	testCases := []struct {
		name       string
		req        *pb.CreateUserRequest
		buildStubs func(ctx context.Context, store *mockdb.MockStore, validator *mockjwt.MockValidator,
			prov *mockprov.MockGitHubProviderService) context.Context
		checkResponse      func(t *testing.T, res *pb.CreateUserResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.CreateUserRequest{},
			buildStubs: func(ctx context.Context, store *mockdb.MockStore, jwt *mockjwt.MockValidator,
				_ *mockprov.MockGitHubProviderService) context.Context {
				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().
					CreateProjectWithID(gomock.Any(), gomock.Any()).
					Return(db.Project{
						ID:   projectID,
						Name: "subject1",
					}, nil)
				store.EXPECT().CreateProvider(gomock.Any(), gomock.Any())

				returnedUser := db.User{
					ID:              1,
					IdentitySubject: "subject1",
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(returnedUser, nil)
				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
				tokenResult, _ := openid.NewBuilder().GivenName("Foo").FamilyName("Bar").Email("test@stacklok.com").Subject("subject1").Build()
				jwt.EXPECT().ParseAndValidate(gomock.Any()).Return(tokenResult, nil)

				return ctx
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, int32(1), res.Id)
				assert.Equal(t, projectID.String(), res.ProjectId)
				assert.Equal(t, "subject1", res.ProjectName)
				assert.NotNil(t, res.CreatedAt)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "Success with pending App",
			req:  &pb.CreateUserRequest{},
			buildStubs: func(ctx context.Context, store *mockdb.MockStore, validator *mockjwt.MockValidator,
				prov *mockprov.MockGitHubProviderService) context.Context {
				ctx = jwt.WithAuthTokenContext(ctx, keyCloakUserToken)

				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)

				returnedUser := db.User{
					ID:              1,
					IdentitySubject: "subject1",
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(returnedUser, nil)

				store.EXPECT().
					GetUnclaimedInstallationsByUser(gomock.Any(), sql.NullString{String: "31337", Valid: true}).
					Return([]db.ProviderGithubAppInstallation{
						{
							AppInstallationID: 10,
							OrganizationID:    9000,
							EnrollingUserID:   sql.NullString{String: "31337", Valid: true},
						},
					}, nil)

				prov.EXPECT().
					CreateGitHubAppWithoutInvitation(gomock.Any(), gomock.Any(), int64(31337), int64(10)).
					Return(&db.Project{
						ID:   projectID,
						Name: "github-org1",
					}, nil)

				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
				tokenResult, _ := openid.NewBuilder().GivenName("Foo").FamilyName("Bar").Email("test@stacklok.com").Subject("subject1").Build()
				validator.EXPECT().ParseAndValidate(gomock.Any()).Return(tokenResult, nil)

				return ctx
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, int32(1), res.Id)
				assert.Equal(t, projectID.String(), res.ProjectId)
				assert.Equal(t, "github-org1", res.ProjectName)
				assert.NotNil(t, res.CreatedAt)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "Success with two pending Apps",
			req:  &pb.CreateUserRequest{},
			buildStubs: func(ctx context.Context, store *mockdb.MockStore, validator *mockjwt.MockValidator,
				prov *mockprov.MockGitHubProviderService) context.Context {
				ctx = jwt.WithAuthTokenContext(ctx, keyCloakUserToken)

				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)

				returnedUser := db.User{
					ID:              1,
					IdentitySubject: "subject1",
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(returnedUser, nil)

				store.EXPECT().
					GetUnclaimedInstallationsByUser(gomock.Any(), sql.NullString{String: "31337", Valid: true}).
					Return([]db.ProviderGithubAppInstallation{
						{
							AppInstallationID: 10,
							OrganizationID:    9000,
							EnrollingUserID:   sql.NullString{String: "31337", Valid: true},
						}, {
							AppInstallationID: 11,
							OrganizationID:    9001,
							EnrollingUserID:   sql.NullString{String: "31337", Valid: true},
						},
					}, nil)

				prov.EXPECT().
					CreateGitHubAppWithoutInvitation(gomock.Any(), gomock.Any(), int64(31337), int64(10)).
					Return(&db.Project{
						ID:   projectID,
						Name: "github-org1",
					}, nil)

				prov.EXPECT().
					CreateGitHubAppWithoutInvitation(gomock.Any(), gomock.Any(), int64(31337), int64(11)).
					Return(&db.Project{
						ID:   uuid.New(),
						Name: "github-org2",
					}, nil)

				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
				tokenResult, _ := openid.NewBuilder().GivenName("Foo").FamilyName("Bar").Email("test@stacklok.com").Subject("subject1").Build()
				validator.EXPECT().ParseAndValidate(gomock.Any()).Return(tokenResult, nil)

				return ctx
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, int32(1), res.Id)
				assert.Equal(t, projectID.String(), res.ProjectId)
				assert.Equal(t, "github-org1", res.ProjectName)
				assert.NotNil(t, res.CreatedAt)
			},
			expectedStatusCode: codes.OK,
		},
	}

	// Create header metadata
	md := metadata.New(map[string]string{
		"authorization": "bearer some-access-token",
	})

	// Create a new context with added header metadata
	ctx := metadata.NewIncomingContext(context.Background(), md)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			mockJwtValidator := mockjwt.NewMockValidator(ctrl)
			mockProviders := mockprov.NewMockGitHubProviderService(ctrl)
			reqCtx := tc.buildStubs(ctx, mockStore, mockJwtValidator, mockProviders)
			crypeng := mockcrypto.NewMockEngine(ctrl)

			authz := &mock.NoopClient{Authorized: true}
			server := &Server{
				store:        mockStore,
				cfg:          &serverconfig.Config{},
				cryptoEngine: crypeng,
				jwt:          mockJwtValidator,
				ghProviders:  mockProviders,
				authzClient:  authz,
				projectCreator: projects.NewProjectCreator(
					authz,
					marketplaces.NewNoopMarketplace(),
					&serverconfig.DefaultProfilesConfig{},
				),
			}

			// server, err := NewServer(mockStore, evt, &serverconfig.Config{
			// 	Auth: serverconfig.AuthConfig{
			// 		TokenKey: generateTokenKey(t),
			// 	},
			// }, mockJwtValidator, ghProviders.NewProviderStore(mockStore))
			// require.NoError(t, err, "failed to create test server")

			resp, err := server.CreateUser(reqCtx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestDeleteUserDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockJwtValidator := mockjwt.NewMockValidator(ctrl)

	request := &pb.DeleteUserRequest{}

	// Create header metadata
	md := metadata.New(map[string]string{
		"authorization": "bearer some-access-token",
	})

	// Create a new context with added header metadata
	ctx := metadata.NewIncomingContext(context.Background(), md)

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case tokenEndpoint:
			data := oauth2.Token{
				AccessToken: "some-token",
			}
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			err := json.NewEncoder(w).Encode(data)
			if err != nil {
				t.Fatal(err)
			}
		case "/admin/realms/stacklok/users/subject1":
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("Unexpected call to mock server endpoint %s", r.URL.Path)
		}
	}))
	defer testServer.Close()

	tokenResult, _ := openid.NewBuilder().Subject("subject1").Build()
	mockJwtValidator.EXPECT().ParseAndValidate(gomock.Any()).Return(tokenResult, nil)

	tx := sql.Tx{}
	mockStore.EXPECT().BeginTransaction().Return(&tx, nil)
	mockStore.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(mockStore)
	mockStore.EXPECT().
		GetUserBySubject(gomock.Any(), "subject1").
		Return(db.User{IdentitySubject: "subject1"}, nil)
	mockStore.EXPECT().
		DeleteUser(gomock.Any(), gomock.Any()).
		Return(nil)
	mockStore.EXPECT().Commit(gomock.Any())
	// we expect rollback to be called even if there is no error (through defer), in that case it will be a no-op
	mockStore.EXPECT().Rollback(gomock.Any())

	crypeng := mockcrypto.NewMockEngine(ctrl)

	server := &Server{
		store: mockStore,
		cfg: &serverconfig.Config{
			Identity: serverconfig.IdentityConfigWrapper{
				Server: serverconfig.IdentityConfig{
					IssuerUrl:    testServer.URL,
					ClientId:     "client-id",
					ClientSecret: "client-secret",
				},
			},
		},
		jwt:          mockJwtValidator,
		cryptoEngine: crypeng,
		authzClient:  &mock.NoopClient{Authorized: true},
	}

	response, err := server.DeleteUser(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestDeleteUser_gRPC(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		req                *pb.DeleteUserRequest
		buildStubs         func(store *mockdb.MockStore, jwt *mockjwt.MockValidator)
		checkResponse      func(t *testing.T, res *pb.DeleteUserResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.DeleteUserRequest{},
			buildStubs: func(store *mockdb.MockStore, jwt *mockjwt.MockValidator) {
				tokenResult, _ := openid.NewBuilder().Subject("subject1").Build()
				jwt.EXPECT().ParseAndValidate(gomock.Any()).Return(tokenResult, nil)

				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().
					GetUserBySubject(gomock.Any(), "subject1").
					Return(db.User{
						IdentitySubject: "subject1",
					}, nil)
				store.EXPECT().
					DeleteUser(gomock.Any(), gomock.Any()).
					Return(nil)
				store.EXPECT().Commit(gomock.Any())
				// we expect rollback to be called even if there is no error (through defer), in that case it will be a no-op
				store.EXPECT().Rollback(gomock.Any())
			},
			checkResponse: func(t *testing.T, res *pb.DeleteUserResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, &pb.DeleteUserResponse{}, res)
			},
			expectedStatusCode: codes.OK,
		},
	}

	// Create header metadata
	md := metadata.New(map[string]string{
		"authorization": "bearer some-access-token",
	})

	// Create a new context with added header metadata
	ctx := metadata.NewIncomingContext(context.Background(), md)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			mockJwtValidator := mockjwt.NewMockValidator(ctrl)
			tc.buildStubs(mockStore, mockJwtValidator)

			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case tokenEndpoint:
					data := oauth2.Token{
						AccessToken: "some-token",
					}
					w.Header().Add("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					err := json.NewEncoder(w).Encode(data)
					if err != nil {
						t.Fatal(err)
					}
				case "/admin/realms/stacklok/users/subject1":
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Fatalf("Unexpected call to mock server endpoint %s", r.URL.Path)
				}
			}))
			defer testServer.Close()

			evt, err := events.Setup(context.Background(), &serverconfig.EventConfig{
				Driver:    "go-channel",
				GoChannel: serverconfig.GoChannelEventConfig{},
			})
			require.NoError(t, err, "failed to setup eventer")

			server := &Server{
				evt:           evt,
				store:         mockStore,
				jwt:           mockJwtValidator,
				providerStore: providers.NewProviderStore(mockStore),
				authzClient:   &mock.SimpleClient{},
				cfg: &serverconfig.Config{
					Auth: serverconfig.AuthConfig{
						TokenKey: generateTokenKey(t),
					},
					Identity: serverconfig.IdentityConfigWrapper{
						Server: serverconfig.IdentityConfig{
							IssuerUrl:    testServer.URL,
							ClientId:     "client-id",
							ClientSecret: "client-secret",
						},
					},
				},
			}

			resp, err := server.DeleteUser(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
