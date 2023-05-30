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

	"github.com/stacklok/mediator/pkg/db"
	"github.com/stretchr/testify/assert"

	"google.golang.org/grpc/codes"

	mockdb "github.com/stacklok/mediator/database/mock"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func TestCreateUserDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	name := "Foo"
	lastname := "Bar"
	request := &pb.CreateUserRequest{
		RoleId:    1,
		Email:     "test@stacklok.com",
		Username:  "test",
		Password:  "1234567@",
		FirstName: &name,
		LastName:  &lastname,
	}

	expectedUser := db.User{
		ID:          1,
		RoleID:      1,
		Email:       "test@stacklok.com",
		Username:    "test",
		Password:    "1234567@",
		FirstName:   sql.NullString{String: "Foo", Valid: true},
		LastName:    sql.NullString{String: "Bar", Valid: true},
		IsProtected: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockStore.EXPECT().
		CreateUser(gomock.Any(), gomock.Any()).
		Return(expectedUser, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.CreateUser(context.Background(), request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedUser.ID, response.Id)
	assert.Equal(t, expectedUser.Username, response.Username)
	assert.Equal(t, expectedUser.Email, response.Email)
	assert.Equal(t, expectedUser.RoleID, response.RoleId)
	assert.Equal(t, expectedUser.IsProtected, *response.IsProtected)
	assert.Equal(t, expectedUser.FirstName, sql.NullString{String: *response.FirstName, Valid: true})
	assert.Equal(t, expectedUser.LastName, sql.NullString{String: *response.LastName, Valid: true})
	expectedCreatedAt := expectedUser.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedUser.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.UpdatedAt.AsTime().In(time.UTC))
}

func TestCreateUser_gRPC(t *testing.T) {
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
				RoleId:   1,
				Username: "test",
				Email:    "test@stacklok.com",
				Password: "1234567@",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Return(db.User{
						ID:          1,
						RoleID:      1,
						Username:    "test",
						Email:       "test@stacklok.com",
						Password:    "1234567@",
						IsProtected: false,
						CreatedAt:   time.Now(),
						UpdatedAt:   time.Now(),
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.CreateUserResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, int32(1), res.Id)
				assert.Equal(t, "test", res.Username)
				assert.Equal(t, "test@stacklok.com", res.Email)
				assert.Equal(t, int32(1), res.RoleId)
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

			resp, err := server.CreateUser(context.Background(), tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
