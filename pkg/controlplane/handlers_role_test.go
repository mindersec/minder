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
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/stacklok/mediator/pkg/auth"
	"github.com/stacklok/mediator/pkg/db"
	"github.com/stretchr/testify/assert"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	mockdb "github.com/stacklok/mediator/database/mock"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func TestCreateRoleDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.CreateRoleRequest{
		GroupId:     1,
		Name:        "TestRole",
		IsAdmin:     nil,
		IsProtected: nil,
	}

	expectedRole := db.Role{
		ID:          1,
		GroupID:     1,
		Name:        "TestRole",
		IsAdmin:     false,
		IsProtected: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), TokenInfoKey, auth.UserClaims{
		UserId:       1,
		IsAdmin:      true,
		IsSuperadmin: true,
	})

	mockStore.EXPECT().
		CreateRole(ctx, gomock.Any()).
		Return(expectedRole, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.CreateRole(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedRole.ID, response.Id)
	assert.Equal(t, expectedRole.Name, response.Name)
	assert.Equal(t, expectedRole.GroupID, response.GroupId)
	assert.Equal(t, expectedRole.IsAdmin, response.IsAdmin)
	assert.Equal(t, expectedRole.IsProtected, response.IsProtected)
	expectedCreatedAt := expectedRole.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedRole.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.UpdatedAt.AsTime().In(time.UTC))
}

func TestCreateRole_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.CreateRoleRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.CreateRoleResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.CreateRoleRequest{
				GroupId: 1,
				Name:    "TestRole",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateRole(gomock.Any(), gomock.Any()).
					Return(db.Role{
						ID:          1,
						GroupID:     1,
						Name:        "TestRole",
						IsAdmin:     false,
						IsProtected: false,
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.CreateRoleResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, int32(1), res.Id)
				assert.Equal(t, "TestRole", res.Name)
				assert.Equal(t, int32(1), res.GroupId)
				assert.Equal(t, false, res.IsAdmin)
				assert.Equal(t, false, res.IsProtected)
				assert.NotNil(t, res.CreatedAt)
				assert.NotNil(t, res.UpdatedAt)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req: &pb.CreateRoleRequest{
				Name: "",
			},
			buildStubs: func(store *mockdb.MockStore) {
				// No expectations, as CreateRole should not be called
			},
			checkResponse: func(t *testing.T, res *pb.CreateRoleResponse, err error) {
				// Assert the expected behavior when the request is empty
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.InvalidArgument,
		},
		{
			name: "StoreError",
			req: &pb.CreateRoleRequest{
				GroupId: 1,
				Name:    "TestRole",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateRole(gomock.Any(), gomock.Any()).
					Return(db.Role{}, errors.New("store error")).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.CreateRoleResponse, err error) {
				// Assert the expected behavior when there's a store error
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.Internal,
		},
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), TokenInfoKey, auth.UserClaims{
		UserId:       1,
		IsAdmin:      true,
		IsSuperadmin: true,
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := NewServer(mockStore)

			resp, err := server.CreateRole(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestDeleteRoleDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.DeleteRoleRequest{Id: 1}

	expectedRole := db.Role{
		ID:          1,
		GroupID:     1,
		Name:        "test",
		IsAdmin:     false,
		IsProtected: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), TokenInfoKey, auth.UserClaims{
		UserId:       1,
		IsAdmin:      true,
		IsSuperadmin: true,
	})

	mockStore.EXPECT().
		GetRoleByID(ctx, gomock.Any()).
		Return(expectedRole, nil)
	mockStore.EXPECT().
		ListUsersByRoleID(ctx, gomock.Any()).
		Return([]db.User{}, nil)
	mockStore.EXPECT().
		DeleteRole(ctx, gomock.Any()).
		Return(nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.DeleteRole(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestDeleteRole_gRPC(t *testing.T) {
	force := true

	testCases := []struct {
		name               string
		req                *pb.DeleteRoleRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.DeleteRoleResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.DeleteRoleRequest{
				Id:    1,
				Force: &force,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetRoleByID(gomock.Any(), gomock.Any()).Return(db.Role{}, nil).Times(1)
				store.EXPECT().
					DeleteRole(gomock.Any(), gomock.Any()).Return(nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.DeleteRoleResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, &pb.DeleteRoleResponse{}, res)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req: &pb.DeleteRoleRequest{
				Id: 0,
			},
			buildStubs: func(store *mockdb.MockStore) {
				// No expectations, as CreateRole should not be called
			},
			checkResponse: func(t *testing.T, res *pb.DeleteRoleResponse, err error) {
				// Assert the expected behavior when the request is empty
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.InvalidArgument,
		},
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), TokenInfoKey, auth.UserClaims{
		UserId:       1,
		IsAdmin:      true,
		IsSuperadmin: true,
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := NewServer(mockStore)

			resp, err := server.DeleteRole(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetRolesDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetRolesRequest{GroupId: 1}

	expectedRoles := []db.Role{
		{
			ID:        1,
			GroupID:   1,
			Name:      "test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:          2,
			GroupID:     1,
			Name:        "test1",
			IsProtected: true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), TokenInfoKey, auth.UserClaims{
		UserId:       1,
		IsAdmin:      true,
		IsSuperadmin: true,
	})

	mockStore.EXPECT().ListRoles(ctx, gomock.Any()).
		Return(expectedRoles, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetRoles(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, len(expectedRoles), len(response.Roles))
	assert.Equal(t, expectedRoles[0].ID, response.Roles[0].Id)
	assert.Equal(t, expectedRoles[0].GroupID, response.Roles[0].GroupId)
	assert.Equal(t, expectedRoles[0].Name, response.Roles[0].Name)

	expectedCreatedAt := expectedRoles[0].CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Roles[0].CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedRoles[0].UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Roles[0].UpdatedAt.AsTime().In(time.UTC))
}

func TestGetRoles_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.GetRolesRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetRolesResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetRolesRequest{GroupId: 1},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListRoles(gomock.Any(), gomock.Any()).
					Return([]db.Role{
						{
							ID:        1,
							GroupID:   1,
							Name:      "test",
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						{
							ID:          2,
							GroupID:     1,
							Name:        "test1",
							IsProtected: true,
							CreatedAt:   time.Now(),
							UpdatedAt:   time.Now(),
						},
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetRolesResponse, err error) {
				expectedRoles := []*pb.RoleRecord{
					{
						Id:        1,
						GroupId:   1,
						Name:      "test",
						CreatedAt: timestamppb.New(time.Now()),
						UpdatedAt: timestamppb.New(time.Now()),
					},
					{
						Id:          2,
						GroupId:     1,
						Name:        "test1",
						IsProtected: true,
						CreatedAt:   timestamppb.New(time.Now()),
						UpdatedAt:   timestamppb.New(time.Now()),
					},
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, len(expectedRoles), len(res.Roles))
				assert.Equal(t, expectedRoles[0].Id, res.Roles[0].Id)
				assert.Equal(t, expectedRoles[0].GroupId, res.Roles[0].GroupId)
				assert.Equal(t, expectedRoles[0].Name, res.Roles[0].Name)
			},
			expectedStatusCode: codes.OK,
		},
	}

	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), TokenInfoKey, auth.UserClaims{
		UserId:       1,
		IsAdmin:      true,
		IsSuperadmin: true,
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := NewServer(mockStore)

			resp, err := server.GetRoles(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetRoleDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetRoleByIdRequest{Id: 1}

	expectedRole := db.Role{
		ID:        1,
		GroupID:   1,
		Name:      "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), TokenInfoKey, auth.UserClaims{
		UserId:       1,
		IsAdmin:      true,
		IsSuperadmin: true,
	})

	mockStore.EXPECT().GetRoleByID(ctx, gomock.Any()).
		Return(expectedRole, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetRoleById(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedRole.ID, response.Role.Id)
	assert.Equal(t, expectedRole.GroupID, response.Role.GroupId)
	assert.Equal(t, expectedRole.Name, response.Role.Name)
	expectedCreatedAt := expectedRole.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Role.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedRole.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Role.UpdatedAt.AsTime().In(time.UTC))
}

func TestGetNonExistingRoleDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetRoleByIdRequest{Id: 5}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), TokenInfoKey, auth.UserClaims{
		UserId:       1,
		IsAdmin:      true,
		IsSuperadmin: true,
	})

	mockStore.EXPECT().GetRoleByID(ctx, gomock.Any()).
		Return(db.Role{}, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetRoleById(ctx, request)

	assert.NoError(t, err)
	assert.Equal(t, int32(0), response.Role.Id)
}

func TestGetRole_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.GetRoleByIdRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetRoleByIdResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetRoleByIdRequest{Id: 1},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetRoleByID(gomock.Any(), gomock.Any()).
					Return(db.Role{
						ID:        1,
						GroupID:   1,
						Name:      "test",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetRoleByIdResponse, err error) {
				expectedRole := pb.RoleRecord{
					Id:        1,
					GroupId:   1,
					Name:      "test",
					CreatedAt: timestamppb.New(time.Now()),
					UpdatedAt: timestamppb.New(time.Now()),
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, expectedRole.Id, res.Role.Id)
				assert.Equal(t, expectedRole.GroupId, res.Role.GroupId)
				assert.Equal(t, expectedRole.Name, res.Role.Name)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "NonExisting",
			req:  &pb.GetRoleByIdRequest{Id: 5},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetRoleByID(gomock.Any(), gomock.Any()).
					Return(db.Role{}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetRoleByIdResponse, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int32(0), res.Role.Id)
			},
			expectedStatusCode: codes.OK,
		},
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), TokenInfoKey, auth.UserClaims{
		UserId:       1,
		IsAdmin:      true,
		IsSuperadmin: true,
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := NewServer(mockStore)

			resp, err := server.GetRoleById(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
