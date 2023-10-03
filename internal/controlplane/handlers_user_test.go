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
	"google.golang.org/protobuf/types/known/timestamppb"

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
		{
			name: "Success 2",
			req:  &pb.CreateUserRequest{},
			buildStubs: func(store *mockdb.MockStore, jwt *mockjwt.MockJwtValidator) {
				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				returnedUser := db.User{
					ID:              1,
					OrganizationID:  rootOrganization,
					IdentitySubject: "subject1",
					Email:           sql.NullString{Valid: false},
					FirstName:       sql.NullString{Valid: false},
					LastName:        sql.NullString{Valid: false},
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}
				store.EXPECT().
					CreateUser(gomock.Any(), db.CreateUserParams{
						OrganizationID:  rootOrganization,
						IdentitySubject: "subject1",
					}).
					Return(returnedUser, nil)
				store.EXPECT().AddUserProject(gomock.Any(), db.AddUserProjectParams{UserID: 1, ProjectID: rootProject})
				store.EXPECT().AddUserRole(gomock.Any(), db.AddUserRoleParams{UserID: 1, RoleID: 1})
				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
				roles := []interface{}{"superadmin"}
				realmAccess := map[string]interface{}{"roles": roles}
				tokenResult, _ := openid.NewBuilder().Subject("subject1").Claim("realm_access", realmAccess).Build()
				jwt.EXPECT().ParseAndValidate(gomock.Any()).Return(tokenResult, nil)
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				t.Helper()

				assert.Equal(t, int32(1), res.Id)
				assert.Equal(t, rootOrganization.String(), res.OrganizationId)
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
				assert.NotNil(t, res.CreatedAt)
				assert.NotNil(t, res.UpdatedAt)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "Superadmin",
			req:  &pb.CreateUserRequest{},
			buildStubs: func(store *mockdb.MockStore, jwt *mockjwt.MockJwtValidator) {
				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(db.User{
						ID:             1,
						OrganizationID: rootOrganization,
						CreatedAt:      time.Now(),
						UpdatedAt:      time.Now(),
					}, nil)
				store.EXPECT().AddUserProject(gomock.Any(), db.AddUserProjectParams{
					UserID:    1,
					ProjectID: rootProject,
				})
				store.EXPECT().AddUserRole(gomock.Any(), db.AddUserRoleParams{UserID: 1, RoleID: 1})
				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
				roles := []interface{}{"superadmin"}
				realmAccess := map[string]interface{}{"roles": roles}
				tokenResult, _ := openid.NewBuilder().Claim("realm_access", realmAccess).Build()
				jwt.EXPECT().ParseAndValidate(gomock.Any()).Return(tokenResult, nil)
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, int32(1), res.Id)
				assert.Equal(t, rootOrganization.String(), res.OrganizationId)
				assert.NotNil(t, res.CreatedAt)
				assert.NotNil(t, res.UpdatedAt)
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
		OrganizationId: rootOrganization,
		ProjectIds:     []uuid.UUID{rootProject},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &rootProject, OrganizationID: rootOrganization}},
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
		OrganizationId: rootOrganization,
		ProjectIds:     []uuid.UUID{rootProject},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &rootProject, OrganizationID: rootOrganization}},
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

func TestGetUsersDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetUsersRequest{}

	expectedUsers := []db.User{
		{
			ID:        1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        2,
			FirstName: sql.NullString{String: "Foo", Valid: true},
			Email:     sql.NullString{String: "test@stacklok.com", Valid: true},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: rootOrganization,
		ProjectIds:     []uuid.UUID{rootProject},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &rootProject, OrganizationID: rootOrganization}},
	})

	mockStore.EXPECT().ListUsers(ctx, gomock.Any()).
		Return(expectedUsers, nil)

	crypeng := crypto.NewEngine("test")

	server := &Server{
		store: mockStore,
		cfg: &config.Config{
			Salt: config.DefaultConfigForTest().Salt,
		},
		cryptoEngine: crypeng,
	}

	response, err := server.GetUsers(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, len(expectedUsers), len(response.Users))
	assert.Equal(t, expectedUsers[0].ID, response.Users[0].Id)
	assert.Equal(t, expectedUsers[0].OrganizationID.String(), response.Users[0].OrganizationId)
	expectedCreatedAt := expectedUsers[0].CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Users[0].CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedUsers[0].UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Users[0].UpdatedAt.AsTime().In(time.UTC))
}

func TestGetUsers_gRPC(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	projectID := uuid.New()

	testCases := []struct {
		name               string
		req                *pb.GetUsersRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetUsersResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetUsersRequest{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListUsers(gomock.Any(), gomock.Any()).
					Return([]db.User{
						{
							ID:              1,
							IdentitySubject: "subject1",
							OrganizationID:  orgID,
							CreatedAt:       time.Now(),
							UpdatedAt:       time.Now(),
						},
						{
							ID:              2,
							IdentitySubject: "subject2",
							OrganizationID:  orgID,
							Email:           sql.NullString{String: "test1@stacklok.com", Valid: true},
							FirstName:       sql.NullString{String: "Foo", Valid: true},
							CreatedAt:       time.Now(),
							UpdatedAt:       time.Now(),
						},
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetUsersResponse, err error) {
				t.Helper()

				firstNamePtr := "Foo"
				emailPtr := "test1@stacklok.com"
				expectedOrgs := []*pb.UserRecord{
					{
						Id:              1,
						IdentitySubject: "subject1",
						OrganizationId:  orgID.String(),
						CreatedAt:       timestamppb.New(time.Now()),
						UpdatedAt:       timestamppb.New(time.Now()),
					},
					{
						Id:              2,
						IdentitySubject: "subject2",
						OrganizationId:  orgID.String(),
						FirstName:       &firstNamePtr,
						Email:           &emailPtr,
						CreatedAt:       timestamppb.New(time.Now()),
						UpdatedAt:       timestamppb.New(time.Now()),
					},
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, len(expectedOrgs), len(res.Users))
				assert.Equal(t, expectedOrgs[1].Id, res.Users[1].Id)
				assert.Equal(t, expectedOrgs[1].OrganizationId, res.Users[1].OrganizationId)
				assert.Equal(t, expectedOrgs[1].FirstName, res.Users[1].FirstName)
				assert.Equal(t, *expectedOrgs[1].Email, *res.Users[1].Email)
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

			resp, err := server.GetUsers(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetUserDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetUserByIdRequest{UserId: 1}

	orgID := uuid.New()
	projectID := uuid.New()

	expectedUser := db.User{
		ID:             1,
		OrganizationID: orgID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projectID},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projectID, OrganizationID: orgID}},
	})

	mockStore.EXPECT().GetUserByID(ctx, gomock.Any()).
		Return(expectedUser, nil)
	mockStore.EXPECT().GetUserRoles(ctx, gomock.Any())
	mockStore.EXPECT().GetUserProjects(ctx, gomock.Any())

	crypeng := crypto.NewEngine("test")

	server := &Server{
		store: mockStore,
		cfg: &config.Config{
			Salt: config.DefaultConfigForTest().Salt,
		},
		cryptoEngine: crypeng,
	}

	response, err := server.GetUserById(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedUser.ID, response.User.Id)
	assert.Equal(t, expectedUser.OrganizationID.String(), response.User.OrganizationId)
	expectedCreatedAt := expectedUser.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.User.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedUser.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.User.UpdatedAt.AsTime().In(time.UTC))
}

func TestGetNonExistingUserDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetUserByIdRequest{UserId: 5}
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: rootOrganization,
		ProjectIds:     []uuid.UUID{rootProject},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &rootProject, OrganizationID: rootOrganization}},
	})

	mockStore.EXPECT().GetUserByID(ctx, gomock.Any()).
		Return(db.User{}, sql.ErrNoRows)

	crypeng := crypto.NewEngine("test")

	server := &Server{
		store: mockStore,
		cfg: &config.Config{
			Salt: config.DefaultConfigForTest().Salt,
		},
		cryptoEngine: crypeng,
	}

	response, err := server.GetUserById(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
}

func TestGetUser_gRPC(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()

	testCases := []struct {
		name               string
		req                *pb.GetUserByIdRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetUserByIdResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetUserByIdRequest{UserId: 1},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserByID(gomock.Any(), gomock.Any()).
					Return(db.User{
						ID:             1,
						OrganizationID: orgID,
						CreatedAt:      time.Now(),
						UpdatedAt:      time.Now(),
					}, nil).
					Times(1)
				store.EXPECT().GetUserRoles(gomock.Any(), gomock.Any())
				store.EXPECT().GetUserProjects(gomock.Any(), gomock.Any())
			},
			checkResponse: func(t *testing.T, res *pb.GetUserByIdResponse, err error) {
				t.Helper()

				expectedUser := pb.UserRecord{
					Id:             1,
					OrganizationId: orgID.String(),
					CreatedAt:      timestamppb.New(time.Now()),
					UpdatedAt:      timestamppb.New(time.Now()),
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, expectedUser.Id, res.User.Id)
				assert.Equal(t, expectedUser.OrganizationId, res.User.OrganizationId)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "NonExisting",
			req:  &pb.GetUserByIdRequest{UserId: 5},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserByID(gomock.Any(), gomock.Any()).
					Return(db.User{}, nil).
					Times(1)
				store.EXPECT().GetUserRoles(gomock.Any(), gomock.Any())
				store.EXPECT().GetUserProjects(gomock.Any(), gomock.Any())

			},
			checkResponse: func(t *testing.T, res *pb.GetUserByIdResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.Equal(t, int32(0), res.User.Id)
			},
			expectedStatusCode: codes.OK,
		},
	}
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: rootOrganization,
		ProjectIds:     []uuid.UUID{rootProject},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &rootProject, OrganizationID: rootOrganization}},
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

			resp, err := server.GetUserById(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
