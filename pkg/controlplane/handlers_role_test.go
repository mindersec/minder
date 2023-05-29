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

	"github.com/stacklok/mediator/pkg/db"
	"github.com/stretchr/testify/assert"

	"google.golang.org/grpc/codes"

	mockdb "github.com/stacklok/mediator/database/mock"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func TestCreateRoleDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.CreateRoleRequest{
		GroupId: 1,
		Name:    "TestRole",
	}

	expectedRole := db.Role{
		ID:          1,
		GroupID:     1,
		Name:        "TestOrg",
		IsAdmin:     false,
		IsProtected: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockStore.EXPECT().
		CreateRole(gomock.Any(), gomock.Any()).
		Return(expectedRole, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.CreateRole(context.Background(), request)

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
				assert.Equal(t, "TestOrg", res.Name)
				assert.Equal(t, 1, res.GroupId)
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
				Name:    "TestOrg",
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

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := NewServer(mockStore)

			resp, err := server.CreateRole(context.Background(), tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
