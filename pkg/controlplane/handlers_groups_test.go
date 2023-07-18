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
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	mockdb "github.com/stacklok/mediator/database/mock"
	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/pkg/auth"
	"github.com/stacklok/mediator/pkg/db"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func TestCreateGroupDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.CreateGroupRequest{
		OrganizationId: 1,
		Name:           "TestGroup",
		Description:    "TestDescription",
		IsProtected:    nil,
	}

	expectedGroup := db.Group{
		ID:             1,
		OrganizationID: 1,
		Name:           "TestGroup",
		Description:    sql.NullString{String: "TestDescription", Valid: true},
		IsProtected:    false,
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
		CreateGroup(ctx, gomock.Any()).
		Return(expectedGroup, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.CreateGroup(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedGroup.ID, response.GroupId)
	assert.Equal(t, expectedGroup.Name, response.Name)
	assert.Equal(t, expectedGroup.OrganizationID, response.OrganizationId)
	assert.Equal(t, expectedGroup.IsProtected, response.IsProtected)
}

func TestCreateGroup_gRPC(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		req                *pb.CreateGroupRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.CreateGroupResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.CreateGroupRequest{
				OrganizationId: 1,
				Name:           "TestGroup",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateGroup(gomock.Any(), gomock.Any()).
					Return(db.Group{
						ID:             1,
						OrganizationID: 1,
						Name:           "TestGroup",
						IsProtected:    false,
						CreatedAt:      time.Now(),
						UpdatedAt:      time.Now(),
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.CreateGroupResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, int32(1), res.GroupId)
				assert.Equal(t, int32(1), res.OrganizationId)
				assert.Equal(t, "TestGroup", res.Name)
				assert.Equal(t, false, res.IsProtected)
				assert.NotNil(t, res.CreatedAt)
				assert.NotNil(t, res.UpdatedAt)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req: &pb.CreateGroupRequest{
				Name: "",
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(t *testing.T, res *pb.CreateGroupResponse, err error) {
				// Assert the expected behavior when the request is empty
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.InvalidArgument,
		},
		{
			name: "StoreError",
			req: &pb.CreateGroupRequest{
				OrganizationId: 1,
				Name:           "TestGroup",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateGroup(gomock.Any(), gomock.Any()).
					Return(db.Group{}, errors.New("store error")).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.CreateGroupResponse, err error) {
				// Assert the expected behavior when there's a store error
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.Internal,
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

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.CreateGroup(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestDeleteGroupDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.DeleteGroupRequest{Id: 1}

	expectedGroup := db.Group{
		ID:             1,
		OrganizationID: 1,
		Name:           "test",
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
		GetGroupByID(gomock.Any(), gomock.Any()).
		Return(expectedGroup, nil).Times(2)
	mockStore.EXPECT().
		ListRolesByGroupID(ctx, gomock.Any()).
		Return([]db.Role{}, nil)
	mockStore.EXPECT().
		DeleteGroup(ctx, gomock.Any()).
		Return(nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.DeleteGroup(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestDeleteGroup_gRPC(t *testing.T) {
	t.Parallel()

	force := true

	testCases := []struct {
		name               string
		req                *pb.DeleteGroupRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.DeleteGroupResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.DeleteGroupRequest{
				Id:    1,
				Force: &force,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetGroupByID(gomock.Any(), gomock.Any()).Return(db.Group{}, nil).Times(2)
				store.EXPECT().
					DeleteGroup(gomock.Any(), gomock.Any()).Return(nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.DeleteGroupResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, &pb.DeleteGroupResponse{}, res)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req: &pb.DeleteGroupRequest{
				Id: 0,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(t *testing.T, res *pb.DeleteGroupResponse, err error) {
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

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.DeleteGroup(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetGroupsDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	request := &pb.GetGroupsRequest{OrganizationId: 1}

	expectedGroups := []db.Group{
		{
			ID:             1,
			OrganizationID: 1,
			Name:           "test",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		{
			ID:             2,
			OrganizationID: 1,
			Name:           "test1",
			IsProtected:    true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}

	mockStore.EXPECT().ListGroups(ctx, gomock.Any()).
		Return(expectedGroups, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetGroups(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, len(expectedGroups), len(response.Groups))
	assert.Equal(t, expectedGroups[0].ID, response.Groups[0].GroupId)
	assert.Equal(t, expectedGroups[0].OrganizationID, response.Groups[0].GroupId)
	assert.Equal(t, expectedGroups[0].Name, response.Groups[0].Name)

	expectedCreatedAt := expectedGroups[0].CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Groups[0].CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedGroups[0].UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Groups[0].UpdatedAt.AsTime().In(time.UTC))
}

func TestGetGroups_gRPC(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		req                *pb.GetGroupsRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetGroupsResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetGroupsRequest{OrganizationId: 1},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListGroups(gomock.Any(), gomock.Any()).
					Return([]db.Group{
						{
							ID:             1,
							OrganizationID: 1,
							Name:           "test",
							CreatedAt:      time.Now(),
							UpdatedAt:      time.Now(),
						},
						{
							ID:             2,
							OrganizationID: 1,
							Name:           "test1",
							IsProtected:    true,
							CreatedAt:      time.Now(),
							UpdatedAt:      time.Now(),
						},
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetGroupsResponse, err error) {
				expectedGroups := []*pb.GroupRecord{
					{
						GroupId:        1,
						OrganizationId: 1,
						Name:           "test",
						CreatedAt:      timestamppb.New(time.Now()),
						UpdatedAt:      timestamppb.New(time.Now()),
					},
					{
						GroupId:        2,
						OrganizationId: 1,
						Name:           "test1",
						IsProtected:    true,
						CreatedAt:      timestamppb.New(time.Now()),
						UpdatedAt:      timestamppb.New(time.Now()),
					},
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, len(expectedGroups), len(res.Groups))
				assert.Equal(t, expectedGroups[0].OrganizationId, res.Groups[0].OrganizationId)
				assert.Equal(t, expectedGroups[0].GroupId, res.Groups[0].GroupId)
				assert.Equal(t, expectedGroups[0].Name, res.Groups[0].Name)
			},
			expectedStatusCode: codes.OK,
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create a new context and set the claims value
			ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
				UserId:         1,
				OrganizationId: 1,
				GroupIds:       []int32{1},
				Roles: []auth.RoleInfo{
					{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
			})

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.GetGroups(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetGroupDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetGroupByIdRequest{GroupId: 1}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	expectedGroup := db.Group{
		ID:             1,
		OrganizationID: 1,
		Name:           "test",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	mockStore.EXPECT().GetGroupByID(ctx, gomock.Any()).
		Return(expectedGroup, nil)
	mockStore.EXPECT().ListRolesByGroupID(ctx, gomock.Any())
	mockStore.EXPECT().ListUsersByGroup(ctx, gomock.Any())

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetGroupById(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedGroup.ID, response.Group.GroupId)
	assert.Equal(t, expectedGroup.OrganizationID, response.Group.OrganizationId)
	assert.Equal(t, expectedGroup.Name, response.Group.Name)
	expectedCreatedAt := expectedGroup.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Group.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedGroup.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Group.UpdatedAt.AsTime().In(time.UTC))
}

func TestGetNonExistingGroupDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetGroupByIdRequest{GroupId: 5}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	mockStore.EXPECT().GetGroupByID(ctx, gomock.Any()).
		Return(db.Group{}, nil)
	mockStore.EXPECT().ListRolesByGroupID(ctx, gomock.Any())
	mockStore.EXPECT().ListUsersByGroup(ctx, gomock.Any())

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetGroupById(ctx, request)

	assert.NoError(t, err)
	assert.Equal(t, int32(0), response.Group.GroupId)
}

func TestGetGroup_gRPC(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		req                *pb.GetGroupByIdRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetGroupByIdResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetGroupByIdRequest{GroupId: 1},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListRolesByGroupID(gomock.Any(), gomock.Any())
				store.EXPECT().ListUsersByGroup(gomock.Any(), gomock.Any())

				store.EXPECT().GetGroupByID(gomock.Any(), gomock.Any()).
					Return(db.Group{
						ID:             1,
						OrganizationID: 1,
						Name:           "test",
						CreatedAt:      time.Now(),
						UpdatedAt:      time.Now(),
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetGroupByIdResponse, err error) {
				expectedGroup := pb.GroupRecord{
					GroupId:        1,
					OrganizationId: 1,
					Name:           "test",
					CreatedAt:      timestamppb.New(time.Now()),
					UpdatedAt:      timestamppb.New(time.Now()),
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, expectedGroup.OrganizationId, res.Group.OrganizationId)
				assert.Equal(t, expectedGroup.GroupId, res.Group.GroupId)
				assert.Equal(t, expectedGroup.Name, res.Group.Name)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "NonExisting",
			req:  &pb.GetGroupByIdRequest{GroupId: 5},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetGroupByID(gomock.Any(), gomock.Any()).
					Return(db.Group{}, nil).
					Times(1)
				store.EXPECT().ListRolesByGroupID(gomock.Any(), gomock.Any())
				store.EXPECT().ListUsersByGroup(gomock.Any(), gomock.Any())

			},
			checkResponse: func(t *testing.T, res *pb.GetGroupByIdResponse, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int32(0), res.Group.GroupId)
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

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.GetGroupById(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
