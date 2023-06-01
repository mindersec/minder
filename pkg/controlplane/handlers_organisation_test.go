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
	"google.golang.org/protobuf/types/known/timestamppb"

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

func TestGetOrganisationsDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetOrganisationsRequest{}

	expectedOrgs := []db.Organisation{
		{
			ID:        1,
			Name:      "TestOrg",
			Company:   "TestCompany",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        2,
			Name:      "TestOrg1",
			Company:   "TestCompany1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	mockStore.EXPECT().ListOrganisations(gomock.Any(), gomock.Any()).
		Return(expectedOrgs, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetOrganisations(context.Background(), request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, len(expectedOrgs), len(response.Organisations))
	assert.Equal(t, expectedOrgs[0].ID, response.Organisations[0].Id)
	assert.Equal(t, expectedOrgs[0].Name, response.Organisations[0].Name)
	assert.Equal(t, expectedOrgs[0].Company, response.Organisations[0].Company)
	expectedCreatedAt := expectedOrgs[0].CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Organisations[0].CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedOrgs[0].UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Organisations[0].UpdatedAt.AsTime().In(time.UTC))
}

func TestGetOrganisations_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.GetOrganisationsRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetOrganisationsResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetOrganisationsRequest{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListOrganisations(gomock.Any(), gomock.Any()).
					Return([]db.Organisation{
						{
							ID:        1,
							Name:      "TestOrg",
							Company:   "TestCompany",
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						{
							ID:        2,
							Name:      "TestOrg1",
							Company:   "TestCompany1",
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetOrganisationsResponse, err error) {
				expectedOrgs := []*pb.OrganisationRecord{
					{
						Id:        1,
						Name:      "TestOrg",
						Company:   "TestCompany",
						CreatedAt: timestamppb.New(time.Now()),
						UpdatedAt: timestamppb.New(time.Now()),
					},
					{
						Id:        2,
						Name:      "TestOrg1",
						Company:   "TestCompany1",
						CreatedAt: timestamppb.New(time.Now()),
						UpdatedAt: timestamppb.New(time.Now()),
					},
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, len(expectedOrgs), len(res.Organisations))
				assert.Equal(t, expectedOrgs[0].Id, res.Organisations[0].Id)
				assert.Equal(t, expectedOrgs[0].Name, res.Organisations[0].Name)
				assert.Equal(t, expectedOrgs[0].Company, res.Organisations[0].Company)
			},
			expectedStatusCode: codes.OK,
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

			resp, err := server.GetOrganisations(context.Background(), tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetOrganisationDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetOrganisationRequest{OrganisationId: 1}

	expectedOrg := db.Organisation{
		ID:        1,
		Name:      "TestOrg",
		Company:   "TestCompany",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockStore.EXPECT().GetOrganisation(gomock.Any(), gomock.Any()).
		Return(expectedOrg, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetOrganisation(context.Background(), request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedOrg.ID, response.Organisation.Id)
	assert.Equal(t, expectedOrg.Name, response.Organisation.Name)
	assert.Equal(t, expectedOrg.Company, response.Organisation.Company)
	expectedCreatedAt := expectedOrg.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Organisation.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedOrg.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Organisation.UpdatedAt.AsTime().In(time.UTC))
}

func TestGetNonExistingOrganisationDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetOrganisationRequest{OrganisationId: 5}

	mockStore.EXPECT().GetOrganisation(gomock.Any(), gomock.Any()).
		Return(db.Organisation{}, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetOrganisation(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, int32(0), response.Organisation.Id)
}

func TestGetOrganisation_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.GetOrganisationRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetOrganisationResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetOrganisationRequest{OrganisationId: 1},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetOrganisation(gomock.Any(), gomock.Any()).
					Return(db.Organisation{
						ID:        1,
						Name:      "TestOrg",
						Company:   "TestCompany",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetOrganisationResponse, err error) {
				expectedOrg := pb.OrganisationRecord{
					Id:        1,
					Name:      "TestOrg",
					Company:   "TestCompany",
					CreatedAt: timestamppb.New(time.Now()),
					UpdatedAt: timestamppb.New(time.Now()),
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, expectedOrg.Id, res.Organisation.Id)
				assert.Equal(t, expectedOrg.Name, res.Organisation.Name)
				assert.Equal(t, expectedOrg.Company, res.Organisation.Company)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "NonExisting",
			req:  &pb.GetOrganisationRequest{OrganisationId: 5},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetOrganisation(gomock.Any(), gomock.Any()).
					Return(db.Organisation{}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetOrganisationResponse, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int32(0), res.Organisation.Id)
			},
			expectedStatusCode: codes.OK,
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

			resp, err := server.GetOrganisation(context.Background(), tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetOrganisationByNameDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetOrganisationByNameRequest{Name: "TestOrg"}

	expectedOrg := db.Organisation{
		ID:        1,
		Name:      "TestOrg",
		Company:   "TestCompany",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockStore.EXPECT().GetOrganisationByName(gomock.Any(), gomock.Any()).
		Return(expectedOrg, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetOrganisationByName(context.Background(), request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedOrg.ID, response.Organisation.Id)
	assert.Equal(t, expectedOrg.Name, response.Organisation.Name)
	assert.Equal(t, expectedOrg.Company, response.Organisation.Company)
	expectedCreatedAt := expectedOrg.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Organisation.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedOrg.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Organisation.UpdatedAt.AsTime().In(time.UTC))
}

func TestGetNonExistingOrganisationByNameDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetOrganisationByNameRequest{Name: "Test"}

	mockStore.EXPECT().GetOrganisationByName(gomock.Any(), gomock.Any()).
		Return(db.Organisation{}, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetOrganisationByName(context.Background(), request)

	assert.NoError(t, err)
	assert.Equal(t, int32(0), response.Organisation.Id)
}

func TestGetOrganisationByName_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.GetOrganisationByNameRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetOrganisationByNameResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetOrganisationByNameRequest{Name: "TestOrg"},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetOrganisationByName(gomock.Any(), gomock.Any()).
					Return(db.Organisation{
						ID:        1,
						Name:      "TestOrg",
						Company:   "TestCompany",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetOrganisationByNameResponse, err error) {
				expectedOrg := pb.OrganisationRecord{
					Id:        1,
					Name:      "TestOrg",
					Company:   "TestCompany",
					CreatedAt: timestamppb.New(time.Now()),
					UpdatedAt: timestamppb.New(time.Now()),
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, expectedOrg.Id, res.Organisation.Id)
				assert.Equal(t, expectedOrg.Name, res.Organisation.Name)
				assert.Equal(t, expectedOrg.Company, res.Organisation.Company)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "NonExisting",
			req:  &pb.GetOrganisationByNameRequest{Name: "test"},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetOrganisationByName(gomock.Any(), gomock.Any()).
					Return(db.Organisation{}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetOrganisationByNameResponse, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int32(0), res.Organisation.Id)
			},
			expectedStatusCode: codes.OK,
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

			resp, err := server.GetOrganisationByName(context.Background(), tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestDeleteOrganisationDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.DeleteOrganisationRequest{Id: 1}

	expectedOrg := db.Organisation{
		ID:        1,
		Name:      "test",
		Company:   "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockStore.EXPECT().GetOrganisation(gomock.Any(), gomock.Any()).
		Return(expectedOrg, nil)
	mockStore.EXPECT().
		ListGroupsByOrganisationID(gomock.Any(), gomock.Any()).
		Return([]db.Group{}, nil)
	mockStore.EXPECT().
		DeleteOrganisation(gomock.Any(), gomock.Any()).
		Return(nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.DeleteOrganisation(context.Background(), request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestDeleteOrganisation_gRPC(t *testing.T) {
	force := true

	testCases := []struct {
		name               string
		req                *pb.DeleteOrganisationRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.DeleteOrganisationResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.DeleteOrganisationRequest{
				Id:    1,
				Force: &force,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetOrganisation(gomock.Any(), gomock.Any()).Return(db.Organisation{}, nil).Times(1)
				store.EXPECT().
					DeleteOrganisation(gomock.Any(), gomock.Any()).Return(nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.DeleteOrganisationResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, &pb.DeleteOrganisationResponse{}, res)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req: &pb.DeleteOrganisationRequest{
				Id: 0,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(t *testing.T, res *pb.DeleteOrganisationResponse, err error) {
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

			resp, err := server.DeleteOrganisation(context.Background(), tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
