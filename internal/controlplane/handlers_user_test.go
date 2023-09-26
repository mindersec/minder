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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	mockdb "github.com/stacklok/mediator/database/mock"
	"github.com/stacklok/mediator/internal/auth"
	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/internal/crypto"
	"github.com/stacklok/mediator/internal/db"
	"github.com/stacklok/mediator/internal/events"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

func TestCreateUserDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	seed := time.Now().UnixNano()

	name := "Foo"
	lastname := "Bar"
	email := "test@stacklok.com"
	password := util.RandomPassword(8, seed)

	request := &pb.CreateUserRequest{
		OrganizationId: 1,
		Email:          &email,
		Username:       "test",
		Password:       &password,
		FirstName:      &name,
		LastName:       &lastname,
	}

	expectedUser := db.User{
		ID:                  1,
		OrganizationID:      1,
		Email:               sql.NullString{String: email, Valid: true},
		Username:            "test",
		Password:            util.RandomPassword(8, seed),
		FirstName:           sql.NullString{String: "Foo", Valid: true},
		LastName:            sql.NullString{String: "Bar", Valid: true},
		IsProtected:         false,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
		NeedsPasswordChange: false,
	}

	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	tx := sql.Tx{}
	mockStore.EXPECT().BeginTransaction().Return(&tx, nil)
	mockStore.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(mockStore)
	mockStore.EXPECT().
		CreateUser(ctx, gomock.Any()).
		Return(expectedUser, nil)
	mockStore.EXPECT().Commit(gomock.Any())
	mockStore.EXPECT().Rollback(gomock.Any())

	crypeng := crypto.NewEngine("test")

	server := &Server{
		store: mockStore,
		cfg: &config.Config{
			Salt: config.DefaultConfigForTest().Salt,
		},
		cryptoEngine: crypeng,
	}

	response, err := server.CreateUser(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedUser.ID, response.Id)
	assert.Equal(t, expectedUser.Username, response.Username)
	assert.Equal(t, expectedUser.Email, sql.NullString{String: *response.Email, Valid: true})
	assert.Equal(t, expectedUser.OrganizationID, response.OrganizationId)
	assert.Equal(t, expectedUser.IsProtected, *response.IsProtected)
	assert.Equal(t, expectedUser.FirstName, sql.NullString{String: *response.FirstName, Valid: true})
	assert.Equal(t, expectedUser.LastName, sql.NullString{String: *response.LastName, Valid: true})
	expectedCreatedAt := expectedUser.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedUser.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.UpdatedAt.AsTime().In(time.UTC))
}

func TestCreateUser_gRPC(t *testing.T) {
	t.Parallel()

	seed := time.Now().UnixNano()
	password := util.RandomPassword(8, seed)

	testCases := []struct {
		name               string
		req                *pb.CreateUserRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.CreateUserResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.CreateUserRequest{
				OrganizationId: 1,
				Username:       "test",
				Password:       &password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				tx := sql.Tx{}
				store.EXPECT().BeginTransaction().Return(&tx, nil)
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)

				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(db.User{
						ID:             1,
						OrganizationID: 1,
						Username:       "test",
						Password:       password,
						IsProtected:    false,
						CreatedAt:      time.Now(),
						UpdatedAt:      time.Now(),
					}, nil).
					Times(1)
				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())

			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, int32(1), res.Id)
				assert.Equal(t, "test", res.Username)
				assert.Equal(t, int32(1), res.OrganizationId)
				assert.Equal(t, false, *res.IsProtected)
				assert.NotNil(t, res.CreatedAt)
				assert.NotNil(t, res.UpdatedAt)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req: &pb.CreateUserRequest{
				Username: "",
			},
			buildStubs: func(store *mockdb.MockStore) {
				// No expectations, as CreateRole should not be called
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				t.Helper()

				// Assert the expected behavior when the request is empty
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.InvalidArgument,
		},
	}

	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)
			evt, err := events.Setup()
			require.NoError(t, err, "failed to setup eventer")
			server, err := NewServer(mockStore, evt, &config.Config{
				Salt: config.DefaultConfigForTest().Salt,
				Auth: config.AuthConfig{
					TokenKey: generateTokenKey(t),
				},
			})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.CreateUser(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestUpdatePasswordDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	seed := time.Now().UnixNano()

	password := util.RandomPassword(8, seed)
	request := &pb.UpdatePasswordRequest{Password: password, PasswordConfirmation: password}

	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	mockStore.EXPECT().GetUserByID(ctx, gomock.Any())
	mockStore.EXPECT().UpdatePassword(ctx, gomock.Any())
	mockStore.EXPECT().RevokeUserToken(ctx, gomock.Any())

	crypeng := crypto.NewEngine("test")

	server := &Server{
		store: mockStore,
		cfg: &config.Config{
			Salt: config.DefaultConfigForTest().Salt,
		},
		cryptoEngine: crypeng,
	}

	response, err := server.UpdatePassword(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestUpdatePassword_gRPC(t *testing.T) {
	t.Parallel()

	seed := time.Now().UnixNano()
	password := util.RandomPassword(8, seed)

	testCases := []struct {
		name               string
		req                *pb.UpdatePasswordRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.UpdatePasswordResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.UpdatePasswordRequest{
				Password:             password,
				PasswordConfirmation: password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserByID(gomock.Any(), gomock.Any())
				store.EXPECT().UpdatePassword(gomock.Any(), gomock.Any())
				store.EXPECT().RevokeUserToken(gomock.Any(), gomock.Any())
			},
			checkResponse: func(t *testing.T, res *pb.UpdatePasswordResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req:  &pb.UpdatePasswordRequest{},
			buildStubs: func(store *mockdb.MockStore) {
				// No expectations, as CreateRole should not be called
			},
			checkResponse: func(t *testing.T, res *pb.UpdatePasswordResponse, err error) {
				t.Helper()

				// Assert the expected behavior when the request is empty
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.InvalidArgument,
		},
	}

	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			evt, err := events.Setup()
			require.NoError(t, err, "failed to setup eventer")
			server, err := NewServer(mockStore, evt, &config.Config{
				Salt: config.DefaultConfigForTest().Salt,
				Auth: config.AuthConfig{
					TokenKey: generateTokenKey(t),
				},
			})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.UpdatePassword(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestUpdateProfileDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	seed := time.Now().UnixNano()

	email := util.RandomEmail(seed)
	name := util.RandomName(seed)

	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	mockStore.EXPECT().UpdateUser(ctx, gomock.Any())

	crypeng := crypto.NewEngine("test")

	server := &Server{
		store: mockStore,
		cfg: &config.Config{
			Salt: config.DefaultConfigForTest().Salt,
		},
		cryptoEngine: crypeng,
	}

	response, err := server.store.UpdateUser(ctx, db.UpdateUserParams{ID: 1, Email: sql.NullString{String: email, Valid: true},
		FirstName: sql.NullString{String: name, Valid: true}})

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestUpdateProfile_gRPC(t *testing.T) {
	t.Parallel()

	seed := time.Now().UnixNano()
	email := util.RandomEmail(seed)
	name := util.RandomName(seed)

	testCases := []struct {
		name               string
		req                *pb.UpdateProfileRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.UpdateProfileResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.UpdateProfileRequest{
				Email:     &email,
				FirstName: &name,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserByID(gomock.Any(), gomock.Any())
				store.EXPECT().UpdateUser(gomock.Any(), gomock.Any())
			},
			checkResponse: func(t *testing.T, res *pb.UpdateProfileResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req:  &pb.UpdateProfileRequest{},
			buildStubs: func(store *mockdb.MockStore) {
				// No expectations, as CreateRole should not be called
			},
			checkResponse: func(t *testing.T, res *pb.UpdateProfileResponse, err error) {
				t.Helper()

				// Assert the expected behavior when the request is empty
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.InvalidArgument,
		},
	}

	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:              1,
		OrganizationId:      1,
		GroupIds:            []int32{1},
		NeedsPasswordChange: false,
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			evt, err := events.Setup()
			require.NoError(t, err, "failed to setup eventer")
			server, err := NewServer(mockStore, evt, &config.Config{
				Salt: config.DefaultConfigForTest().Salt,
				Auth: config.AuthConfig{
					TokenKey: generateTokenKey(t),
				},
			})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.UpdateProfile(ctx, tc.req)
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

	expectedUser := db.User{
		ID:             1,
		OrganizationID: 1,
		Email:          sql.NullString{String: "test@stacklok.com", Valid: true},
		Username:       "test",
		Password:       util.RandomPassword(8, time.Now().UnixNano()),
		FirstName:      sql.NullString{String: "Foo", Valid: true},
		LastName:       sql.NullString{String: "Bar", Valid: true},
		IsProtected:    false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
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
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			evt, err := events.Setup()
			require.NoError(t, err, "failed to setup eventer")
			server, err := NewServer(mockStore, evt, &config.Config{
				Salt: config.DefaultConfigForTest().Salt,
				Auth: config.AuthConfig{
					TokenKey: generateTokenKey(t),
				},
			})
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
			Username:  "test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:          2,
			Username:    "test1",
			FirstName:   sql.NullString{String: "Foo", Valid: true},
			Email:       sql.NullString{String: "test@stacklok.com", Valid: true},
			IsProtected: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
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
	assert.Equal(t, expectedUsers[0].OrganizationID, response.Users[0].OrganizationId)
	assert.Equal(t, expectedUsers[0].Username, response.Users[0].Username)
	expectedCreatedAt := expectedUsers[0].CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Users[0].CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedUsers[0].UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Users[0].UpdatedAt.AsTime().In(time.UTC))
}

func TestGetUsers_gRPC(t *testing.T) {
	t.Parallel()

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
							ID:             1,
							OrganizationID: 1,
							Username:       "test",
							CreatedAt:      time.Now(),
							UpdatedAt:      time.Now(),
						},
						{
							ID:             2,
							OrganizationID: 1,
							Email:          sql.NullString{String: "test1@stacklok.com", Valid: true},
							Username:       "test1",
							FirstName:      sql.NullString{String: "Foo", Valid: true},
							IsProtected:    true,
							CreatedAt:      time.Now(),
							UpdatedAt:      time.Now(),
						},
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetUsersResponse, err error) {
				t.Helper()

				firstNamePtr := "Foo"
				protectedPtr := true
				emailPtr := "test1@stacklok.com"
				expectedOrgs := []*pb.UserRecord{
					{
						Id:             1,
						OrganizationId: 1,
						Username:       "test",
						CreatedAt:      timestamppb.New(time.Now()),
						UpdatedAt:      timestamppb.New(time.Now()),
					},
					{
						Id:             2,
						OrganizationId: 1,
						Username:       "test1",
						FirstName:      &firstNamePtr,
						Email:          &emailPtr,
						IsProtected:    &protectedPtr,
						CreatedAt:      timestamppb.New(time.Now()),
						UpdatedAt:      timestamppb.New(time.Now()),
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
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			evt, err := events.Setup()
			require.NoError(t, err, "failed to setup eventer")
			server, err := NewServer(mockStore, evt, &config.Config{
				Salt: config.DefaultConfigForTest().Salt,
				Auth: config.AuthConfig{
					TokenKey: generateTokenKey(t),
				},
			})
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

	request := &pb.GetUserByIdRequest{Id: 1}

	expectedUser := db.User{
		ID:             1,
		OrganizationID: 1,
		Username:       "test",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	mockStore.EXPECT().GetUserByID(ctx, gomock.Any()).
		Return(expectedUser, nil)
	mockStore.EXPECT().GetUserRoles(ctx, gomock.Any())
	mockStore.EXPECT().GetUserGroups(ctx, gomock.Any())

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
	assert.Equal(t, expectedUser.OrganizationID, response.User.OrganizationId)
	assert.Equal(t, expectedUser.Username, response.User.Username)
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

	request := &pb.GetUserByIdRequest{Id: 5}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	mockStore.EXPECT().GetUserByID(ctx, gomock.Any()).
		Return(db.User{}, nil)
	mockStore.EXPECT().GetUserRoles(ctx, gomock.Any())
	mockStore.EXPECT().GetUserGroups(ctx, gomock.Any())

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
	assert.Equal(t, int32(0), response.User.Id)
}

func TestGetUser_gRPC(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		req                *pb.GetUserByIdRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetUserByIdResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetUserByIdRequest{Id: 1},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserByID(gomock.Any(), gomock.Any()).
					Return(db.User{
						ID:             1,
						OrganizationID: 1,
						Username:       "test",
						CreatedAt:      time.Now(),
						UpdatedAt:      time.Now(),
					}, nil).
					Times(1)
				store.EXPECT().GetUserRoles(gomock.Any(), gomock.Any())
				store.EXPECT().GetUserGroups(gomock.Any(), gomock.Any())
			},
			checkResponse: func(t *testing.T, res *pb.GetUserByIdResponse, err error) {
				t.Helper()

				expectedUser := pb.UserRecord{
					Id:             1,
					OrganizationId: 1,
					Username:       "test",
					CreatedAt:      timestamppb.New(time.Now()),
					UpdatedAt:      timestamppb.New(time.Now()),
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, expectedUser.Id, res.User.Id)
				assert.Equal(t, expectedUser.OrganizationId, res.User.OrganizationId)
				assert.Equal(t, expectedUser.Username, res.User.Username)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "NonExisting",
			req:  &pb.GetUserByIdRequest{Id: 5},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetUserByID(gomock.Any(), gomock.Any()).
					Return(db.User{}, nil).
					Times(1)
				store.EXPECT().GetUserRoles(gomock.Any(), gomock.Any())
				store.EXPECT().GetUserGroups(gomock.Any(), gomock.Any())

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
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			evt, err := events.Setup()
			require.NoError(t, err, "failed to setup eventer")
			server, err := NewServer(mockStore, evt, &config.Config{
				Salt: config.DefaultConfigForTest().Salt,
				Auth: config.AuthConfig{
					TokenKey: generateTokenKey(t),
				},
			})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.GetUserById(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
