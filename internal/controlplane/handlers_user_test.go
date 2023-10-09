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
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	mockdb "github.com/stacklok/mediator/database/mock"
	"github.com/stacklok/mediator/internal/auth"
	mockjwt "github.com/stacklok/mediator/internal/auth/mock"
	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/internal/crypto"
	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/events"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
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
					Email:           sql.NullString{String: "test@stacklok.com", Valid: true},
					FirstName:       sql.NullString{String: "Foo", Valid: true},
					LastName:        sql.NullString{String: "Bar", Valid: true},
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				store.EXPECT().
					CreateOrganization(gomock.Any(), gomock.Any()).
					Return(db.Project{ID: orgID}, nil)
				store.EXPECT().
					CreateProject(gomock.Any(), gomock.Any()).
					Return(db.Project{
						ID: projectID,
						ParentID: uuid.NullUUID{
							UUID:  orgID,
							Valid: true,
						},
					}, nil)
				store.EXPECT().
					CreateRole(gomock.Any(), gomock.Any()).
					Return(db.Role{ID: 2}, nil)
				store.EXPECT().
					CreateRole(gomock.Any(), gomock.Any()).
					Return(db.Role{ID: 3}, nil)
				store.EXPECT().CreateProvider(gomock.Any(), gomock.Any())
				store.EXPECT().
					CreateUser(gomock.Any(), db.CreateUserParams{OrganizationID: orgID,
						IdentitySubject: "subject1",
						Email:           sql.NullString{String: "test@stacklok.com", Valid: true},
						FirstName:       sql.NullString{String: "Foo", Valid: true},
						LastName:        sql.NullString{String: "Bar", Valid: true}}).
					Return(returnedUser, nil)
				store.EXPECT().AddUserProject(gomock.Any(), db.AddUserProjectParams{UserID: 1, ProjectID: projectID})
				store.EXPECT().AddUserRole(gomock.Any(), db.AddUserRoleParams{UserID: 1, RoleID: 2})
				store.EXPECT().AddUserRole(gomock.Any(), db.AddUserRoleParams{UserID: 1, RoleID: 3})
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
				assert.Equal(t, "test@stacklok.com", *res.Email)
				assert.Equal(t, orgID.String(), res.OrganizationId)
				assert.Equal(t, "Foo", *res.FirstName)
				assert.Equal(t, "Bar", *res.LastName)
			},
		},
	}

	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projectID},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projectID, OrganizationID: orgID}},
	})

	// Create header metadata
	md := metadata.New(map[string]string{
		"authorization": "bearer some-access-token",
	})

	// Create a new context with added header metadata
	ctx = metadata.NewIncomingContext(ctx, md)

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
				store: mockStore,
				cfg: &config.Config{
					Salt: config.DefaultConfigForTest().Salt,
				},
				cryptoEngine: crypeng,
				vldtr:        mockJwtValidator,
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
					CreateProject(gomock.Any(), gomock.Any()).
					Return(db.Project{
						ID: projectID,
						ParentID: uuid.NullUUID{
							UUID:  orgID,
							Valid: true,
						},
					}, nil)
				store.EXPECT().
					CreateRole(gomock.Any(), gomock.Any()).
					Return(db.Role{ID: 2}, nil)
				store.EXPECT().
					CreateRole(gomock.Any(), gomock.Any()).
					Return(db.Role{ID: 3}, nil)
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
				store.EXPECT().AddUserProject(gomock.Any(), db.AddUserProjectParams{UserID: 1, ProjectID: projectID})
				store.EXPECT().AddUserRole(gomock.Any(), db.AddUserRoleParams{UserID: 1, RoleID: 2})
				store.EXPECT().AddUserRole(gomock.Any(), db.AddUserRoleParams{UserID: 1, RoleID: 3})
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

	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projectID},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projectID, OrganizationID: orgID}},
	})

	// Create header metadata
	md := metadata.New(map[string]string{
		"authorization": "bearer some-access-token",
	})

	// Create a new context with added header metadata
	ctx = metadata.NewIncomingContext(ctx, md)

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			mockJwtValidator := mockjwt.NewMockJwtValidator(ctrl)
			tc.buildStubs(mockStore, mockJwtValidator)
			evt, err := events.Setup()
			require.NoError(t, err, "failed to setup eventer")
			server, err := NewServer(mockStore, evt, &config.Config{
				Salt: config.DefaultConfigForTest().Salt,
				Auth: config.AuthConfig{
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

	request := &pb.DeleteUserRequest{Id: 1}

	orgID := uuid.New()
	projectID := uuid.New()

	expectedUser := db.User{
		ID:             1,
		OrganizationID: orgID,
		Email:          sql.NullString{String: "test@stacklok.com", Valid: true},
		FirstName:      sql.NullString{String: "Foo", Valid: true},
		LastName:       sql.NullString{String: "Bar", Valid: true},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
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

	mockStore.EXPECT().
		GetUserByID(ctx, gomock.Any()).
		Return(expectedUser, nil)
	mockStore.EXPECT().
		DeleteUser(ctx, gomock.Any()).
		Return(nil)

	crypeng := crypto.NewEngine("test")

	server := &Server{
		store: mockStore,
		cfg: &config.Config{
			Salt: config.DefaultConfigForTest().Salt,
		},
		cryptoEngine: crypeng,
	}

	response, err := server.DeleteUser(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestDeleteUser_gRPC(t *testing.T) {
	t.Parallel()

	force := true

	orgID := uuid.New()
	projectID := uuid.New()

	testCases := []struct {
		name               string
		req                *pb.DeleteUserRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.DeleteUserResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.DeleteUserRequest{
				Id:    1,
				Force: &force,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUserByID(gomock.Any(), gomock.Any()).Return(db.User{}, nil).Times(1)
				store.EXPECT().
					DeleteUser(gomock.Any(), gomock.Any()).Return(nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.DeleteUserResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, &pb.DeleteUserResponse{}, res)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req: &pb.DeleteUserRequest{
				Id: 0,
			},
			buildStubs: func(store *mockdb.MockStore) {
				// No expectations, as CreateRole should not be called
			},
			checkResponse: func(t *testing.T, res *pb.DeleteUserResponse, err error) {
				t.Helper()

				// Assert the expected behavior when the request is empty
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.InvalidArgument,
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

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)
			mockJwtValidator := mockjwt.NewMockJwtValidator(ctrl)

			evt, err := events.Setup()
			require.NoError(t, err, "failed to setup eventer")
			server, err := NewServer(mockStore, evt, &config.Config{
				Salt: config.DefaultConfigForTest().Salt,
				Auth: config.AuthConfig{
					TokenKey: generateTokenKey(t),
				},
			}, mockJwtValidator)
			require.NoError(t, err, "failed to create test server")

			resp, err := server.DeleteUser(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
