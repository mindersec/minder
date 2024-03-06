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
	mockjwt "github.com/stacklok/minder/internal/auth/mock"
	"github.com/stacklok/minder/internal/authz/mock"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/events"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	//nolint:gosec // not credentials, just an endpoint
	tokenEndpoint = "/realms/stacklok/protocol/openid-connect/token"
)

func TestCreateUserDBMock(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	projectID := uuid.New()

	testCases := []struct {
		name               string
		req                *pb.CreateUserRequest
		buildStubs         func(store *mockdb.MockStore, jwt *mockjwt.MockJwtValidator)
		checkResponse      func(t *testing.T, res *pb.CreateUserResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success 1",
			req:  &pb.CreateUserRequest{},
			buildStubs: func(store *mockdb.MockStore, jwt *mockjwt.MockJwtValidator) {
				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				returnedUser := db.User{
					ID:              1,
					OrganizationID:  orgID,
					IdentitySubject: "subject1",
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				store.EXPECT().
					CreateOrganization(gomock.Any(), gomock.Any()).
					Return(db.Project{ID: orgID}, nil)
				store.EXPECT().
					CreateProjectWithID(gomock.Any(), gomock.Any()).
					Return(db.Project{
						ID: projectID,
						ParentID: uuid.NullUUID{
							UUID:  orgID,
							Valid: true,
						},
					}, nil)
				store.EXPECT().CreateProvider(gomock.Any(), gomock.Any())
				store.EXPECT().
					CreateUser(gomock.Any(), db.CreateUserParams{OrganizationID: orgID,
						IdentitySubject: "subject1"}).
					Return(returnedUser, nil)
				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
				tokenResult, _ := openid.NewBuilder().GivenName("Foo").FamilyName("Bar").Email("test@stacklok.com").Subject("subject1").Build()
				jwt.EXPECT().ParseAndValidate(gomock.Any()).Return(tokenResult, nil)
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, int32(1), res.Id)
				assert.Equal(t, orgID.String(), res.OrganizationId)
			},
		},
	}

	// Create header metadata
	md := metadata.New(map[string]string{
		"authorization": "bearer some-access-token",
	})

	// Create a new context with added header metadata
	ctx := metadata.NewIncomingContext(context.Background(), md)

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			mockJwtValidator := mockjwt.NewMockJwtValidator(ctrl)
			tc.buildStubs(mockStore, mockJwtValidator)
			crypeng := crypto.NewEngine("test")

			server := &Server{
				store:        mockStore,
				cfg:          &serverconfig.Config{},
				cryptoEngine: crypeng,
				vldtr:        mockJwtValidator,
				authzClient:  &mock.NoopClient{Authorized: true},
			}

			resp, err := server.CreateUser(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestCreateUser_gRPC(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	projectID := uuid.New()

	testCases := []struct {
		name               string
		req                *pb.CreateUserRequest
		buildStubs         func(store *mockdb.MockStore, jwt *mockjwt.MockJwtValidator)
		checkResponse      func(t *testing.T, res *pb.CreateUserResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.CreateUserRequest{},
			buildStubs: func(store *mockdb.MockStore, jwt *mockjwt.MockJwtValidator) {
				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().
					CreateOrganization(gomock.Any(), gomock.Any()).
					Return(db.Project{ID: orgID}, nil)
				store.EXPECT().
					CreateProjectWithID(gomock.Any(), gomock.Any()).
					Return(db.Project{
						ID: projectID,
						ParentID: uuid.NullUUID{
							UUID:  orgID,
							Valid: true,
						},
					}, nil)
				store.EXPECT().CreateProvider(gomock.Any(), gomock.Any())
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(db.User{
						ID:             1,
						OrganizationID: orgID,
						CreatedAt:      time.Now(),
						UpdatedAt:      time.Now(),
					}, nil).
					Times(1)
				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
				jwt.EXPECT().ParseAndValidate(gomock.Any()).Return(openid.New(), nil)
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, int32(1), res.Id)
				assert.Equal(t, orgID.String(), res.OrganizationId)
				assert.Equal(t, projectID.String(), res.ProjectId)
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

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			mockJwtValidator := mockjwt.NewMockJwtValidator(ctrl)
			tc.buildStubs(mockStore, mockJwtValidator)
			evt, err := events.Setup(context.Background(), &serverconfig.EventConfig{
				Driver:    "go-channel",
				GoChannel: serverconfig.GoChannelEventConfig{},
			})
			require.NoError(t, err, "failed to setup eventer")
			server, err := NewServer(mockStore, evt, &serverconfig.Config{
				Auth: serverconfig.AuthConfig{
					TokenKey: generateTokenKey(t),
				},
			}, mockJwtValidator)
			require.NoError(t, err, "failed to create test server")

			resp, err := server.CreateUser(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestDeleteUserDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockJwtValidator := mockjwt.NewMockJwtValidator(ctrl)

	request := &pb.DeleteUserRequest{}

	orgID := uuid.New()

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
		Return(db.User{
			OrganizationID: orgID,
		}, nil)
	mockStore.EXPECT().
		DeleteOrganization(gomock.Any(), orgID).
		Return(nil)
	mockStore.EXPECT().Commit(gomock.Any())
	mockStore.EXPECT().Rollback(gomock.Any())

	crypeng := crypto.NewEngine("test")

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
		vldtr:        mockJwtValidator,
		cryptoEngine: crypeng,
		authzClient:  &mock.NoopClient{Authorized: true},
	}

	response, err := server.DeleteUser(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestDeleteUser_gRPC(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()

	testCases := []struct {
		name               string
		req                *pb.DeleteUserRequest
		buildStubs         func(store *mockdb.MockStore, jwt *mockjwt.MockJwtValidator)
		checkResponse      func(t *testing.T, res *pb.DeleteUserResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.DeleteUserRequest{},
			buildStubs: func(store *mockdb.MockStore, jwt *mockjwt.MockJwtValidator) {
				tokenResult, _ := openid.NewBuilder().Subject("subject1").Build()
				jwt.EXPECT().ParseAndValidate(gomock.Any()).Return(tokenResult, nil)

				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().
					GetUserBySubject(gomock.Any(), "subject1").
					Return(db.User{
						OrganizationID: orgID,
					}, nil)
				store.EXPECT().
					DeleteOrganization(gomock.Any(), orgID).
					Return(nil)
				store.EXPECT().Commit(gomock.Any())
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

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			mockJwtValidator := mockjwt.NewMockJwtValidator(ctrl)
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
			server, err := NewServer(mockStore, evt, &serverconfig.Config{
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
			}, mockJwtValidator)
			require.NoError(t, err, "failed to create test server")

			resp, err := server.DeleteUser(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
