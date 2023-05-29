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

func TestCreateOrganisationDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.CreateOrganisationRequest{
		Name:    "TestOrg",
		Company: "TestCompany",
	}

	expectedOrg := db.Organisation{
		ID:        1,
		Name:      "TestOrg",
		Company:   "TestCompany",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockStore.EXPECT().
		CreateOrganisation(gomock.Any(), gomock.Any()).
		Return(expectedOrg, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.CreateOrganisation(context.Background(), request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedOrg.ID, response.Id)
	assert.Equal(t, expectedOrg.Name, response.Name)
	assert.Equal(t, expectedOrg.Company, response.Company)
	expectedCreatedAt := expectedOrg.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedOrg.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.UpdatedAt.AsTime().In(time.UTC))
}

func TestCreateOrganisation_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.CreateOrganisationRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.CreateOrganisationResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.CreateOrganisationRequest{
				Name:    "TestOrg",
				Company: "TestCompany",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateOrganisation(gomock.Any(), gomock.Any()).
					Return(db.Organisation{
						ID:        1,
						Name:      "TestOrg",
						Company:   "TestCompany",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.CreateOrganisationResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, int32(1), res.Id)
				assert.Equal(t, "TestOrg", res.Name)
				assert.Equal(t, "TestCompany", res.Company)
				assert.NotNil(t, res.CreatedAt)
				assert.NotNil(t, res.UpdatedAt)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req: &pb.CreateOrganisationRequest{
				Name:    "",
				Company: "",
			},
			buildStubs: func(store *mockdb.MockStore) {
				// No expectations, as CreateOrganisation should not be called
			},
			checkResponse: func(t *testing.T, res *pb.CreateOrganisationResponse, err error) {
				// Assert the expected behavior when the request is empty
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.InvalidArgument,
		},
		{
			name: "StoreError",
			req: &pb.CreateOrganisationRequest{
				Name:    "TestOrg",
				Company: "TestCompany",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateOrganisation(gomock.Any(), gomock.Any()).
					Return(db.Organisation{}, errors.New("store error")).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.CreateOrganisationResponse, err error) {
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

			resp, err := server.CreateOrganisation(context.Background(), tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
