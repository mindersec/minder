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

	"github.com/stacklok/mediator/pkg/db"
	"github.com/stretchr/testify/assert"

	"google.golang.org/grpc/codes"

	mockdb "github.com/stacklok/mediator/database/mock"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func TestCreateGroupDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.CreateGroupRequest{
		OrganisationId: 1,
		Name:           "TestGroup",
		Description:    "TestDescription",
		IsProtected:    nil,
	}

	expectedGroup := db.Group{
		ID:             1,
		OrganisationID: 1,
		Name:           "TestGroup",
		Description:    sql.NullString{String: "TestDescription", Valid: true},
		IsProtected:    false,
	}

	mockStore.EXPECT().
		GetGroupByName(gomock.Any(), gomock.Any()).Return(db.Group{}, sql.ErrNoRows)
	mockStore.EXPECT().
		CreateGroup(gomock.Any(), gomock.Any()).
		Return(expectedGroup, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.CreateGroup(context.Background(), request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedGroup.ID, response.GroupId)
	assert.Equal(t, expectedGroup.Name, response.Name)
	assert.Equal(t, expectedGroup.OrganisationID, response.OrganisationId)
	assert.Equal(t, expectedGroup.IsProtected, response.IsProtected)
}

func TestCreateGroup_gRPC(t *testing.T) {
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
				OrganisationId: 1,
				Name:           "TestRole",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetGroupByName(gomock.Any(), gomock.Any()).Return(db.Group{}, sql.ErrNoRows).Times(1)
				store.EXPECT().
					CreateGroup(gomock.Any(), gomock.Any()).
					Return(db.Group{
						ID:             1,
						OrganisationID: 1,
						Name:           "TestRole",
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
				assert.Equal(t, int32(1), res.OrganisationId)
				assert.Equal(t, "TestRole", res.Name)
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
				// No expectations, as CreateRole should not be called
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
				OrganisationId: 1,
				Name:           "TestRole",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetGroupByName(gomock.Any(), gomock.Any()).Return(db.Group{}, sql.ErrNoRows).Times(1)
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

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := NewServer(mockStore)

			resp, err := server.CreateGroup(context.Background(), tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestDeleteGroupDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.DeleteGroupRequest{Id: 1}

	expectedGroup := db.Group{
		ID:             1,
		OrganisationID: 1,
		Name:           "test",
		IsProtected:    false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	mockStore.EXPECT().
		GetGroupByID(gomock.Any(), gomock.Any()).
		Return(expectedGroup, nil)
	mockStore.EXPECT().
		ListRolesByGroupID(gomock.Any(), gomock.Any()).
		Return([]db.Role{}, nil)
	mockStore.EXPECT().
		DeleteGroup(gomock.Any(), gomock.Any()).
		Return(nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.DeleteGroup(context.Background(), request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestDeleteGroup_gRPC(t *testing.T) {
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
					GetGroupByID(gomock.Any(), gomock.Any()).Return(db.Group{}, nil).Times(1)
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

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := NewServer(mockStore)

			resp, err := server.DeleteGroup(context.Background(), tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
