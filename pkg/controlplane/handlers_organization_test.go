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

	"github.com/stacklok/mediator/internal/config"
	"github.com/stacklok/mediator/pkg/auth"
	"github.com/stacklok/mediator/pkg/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	mockdb "github.com/stacklok/mediator/database/mock"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

func TestCreateOrganizationDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.CreateOrganizationRequest{
		Name:    "TestOrg",
		Company: "TestCompany",
	}

	expectedOrg := db.Organization{
		ID:        1,
		Name:      "TestOrg",
		Company:   "TestCompany",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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
		CreateOrganization(ctx, gomock.Any()).Return(expectedOrg, nil)
	mockStore.EXPECT().Commit(gomock.Any())
	mockStore.EXPECT().Rollback(gomock.Any())

	server := &Server{
		store: mockStore,
	}

	response, err := server.CreateOrganization(ctx, request)

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

func TestCreateOrganization_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.CreateOrganizationRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.CreateOrganizationResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.CreateOrganizationRequest{
				Name:    "TestOrg",
				Company: "TestCompany",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().BeginTransaction()
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)
				store.EXPECT().
					CreateOrganization(gomock.Any(), gomock.Any()).
					Return(db.Organization{
						ID:        1,
						Name:      "TestOrg",
						Company:   "TestCompany",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil).
					Times(1)
				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
			},
			checkResponse: func(t *testing.T, res *pb.CreateOrganizationResponse, err error) {
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
			req: &pb.CreateOrganizationRequest{
				Name:    "",
				Company: "",
			},
			buildStubs: func(store *mockdb.MockStore) {
				// No expectations, as CreateOrganization should not be called
			},
			checkResponse: func(t *testing.T, res *pb.CreateOrganizationResponse, err error) {
				// Assert the expected behavior when the request is empty
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.InvalidArgument,
		},
		{
			name: "StoreError",
			req: &pb.CreateOrganizationRequest{
				Name:    "TestOrg",
				Company: "TestCompany",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().BeginTransaction()
				store.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(store)

				store.EXPECT().
					CreateOrganization(gomock.Any(), gomock.Any()).
					Return(db.Organization{}, errors.New("store error")).
					Times(1)
				store.EXPECT().Rollback(gomock.Any())
			},
			checkResponse: func(t *testing.T, res *pb.CreateOrganizationResponse, err error) {
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

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.CreateOrganization(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetOrganizationsDBMock(t *testing.T) {
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

	request := &pb.GetOrganizationsRequest{}

	expectedOrgs := []db.Organization{
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

	mockStore.EXPECT().ListOrganizations(ctx, gomock.Any()).
		Return(expectedOrgs, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetOrganizations(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, len(expectedOrgs), len(response.Organizations))
	assert.Equal(t, expectedOrgs[0].ID, response.Organizations[0].Id)
	assert.Equal(t, expectedOrgs[0].Name, response.Organizations[0].Name)
	assert.Equal(t, expectedOrgs[0].Company, response.Organizations[0].Company)
	expectedCreatedAt := expectedOrgs[0].CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Organizations[0].CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedOrgs[0].UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Organizations[0].UpdatedAt.AsTime().In(time.UTC))
}

func TestGetOrganizations_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.GetOrganizationsRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetOrganizationsResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetOrganizationsRequest{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListOrganizations(gomock.Any(), gomock.Any()).
					Return([]db.Organization{
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
			checkResponse: func(t *testing.T, res *pb.GetOrganizationsResponse, err error) {
				expectedOrgs := []*pb.OrganizationRecord{
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
				assert.Equal(t, len(expectedOrgs), len(res.Organizations))
				assert.Equal(t, expectedOrgs[0].Id, res.Organizations[0].Id)
				assert.Equal(t, expectedOrgs[0].Name, res.Organizations[0].Name)
				assert.Equal(t, expectedOrgs[0].Company, res.Organizations[0].Company)
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

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.GetOrganizations(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetOrganizationDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetOrganizationRequest{OrganizationId: 1}

	expectedOrg := db.Organization{
		ID:        1,
		Name:      "TestOrg",
		Company:   "TestCompany",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})
	mockStore.EXPECT().GetOrganization(ctx, gomock.Any()).
		Return(expectedOrg, nil)
	mockStore.EXPECT().ListGroupsByOrganizationID(ctx, gomock.Any())
	mockStore.EXPECT().ListRoles(ctx, gomock.Any())
	mockStore.EXPECT().ListUsersByOrganization(ctx, gomock.Any())

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetOrganization(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedOrg.ID, response.Organization.Id)
	assert.Equal(t, expectedOrg.Name, response.Organization.Name)
	assert.Equal(t, expectedOrg.Company, response.Organization.Company)
	expectedCreatedAt := expectedOrg.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Organization.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedOrg.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Organization.UpdatedAt.AsTime().In(time.UTC))
}

func TestGetNonExistingOrganizationDBMock(t *testing.T) {
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

	request := &pb.GetOrganizationRequest{OrganizationId: 5}

	mockStore.EXPECT().GetOrganization(ctx, gomock.Any()).
		Return(db.Organization{}, nil)
	mockStore.EXPECT().ListGroupsByOrganizationID(ctx, gomock.Any())
	mockStore.EXPECT().ListRoles(ctx, gomock.Any())
	mockStore.EXPECT().ListUsersByOrganization(ctx, gomock.Any())

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetOrganization(ctx, request)

	assert.NoError(t, err)
	assert.Equal(t, int32(0), response.Organization.Id)
}

func TestGetOrganization_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.GetOrganizationRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetOrganizationResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetOrganizationRequest{OrganizationId: 1},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetOrganization(gomock.Any(), gomock.Any()).
					Return(db.Organization{
						ID:        1,
						Name:      "TestOrg",
						Company:   "TestCompany",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil).
					Times(1)
				store.EXPECT().ListRoles(gomock.Any(), gomock.Any())
				store.EXPECT().ListGroupsByOrganizationID(gomock.Any(), gomock.Any())
				store.EXPECT().ListUsersByOrganization(gomock.Any(), gomock.Any())

			},
			checkResponse: func(t *testing.T, res *pb.GetOrganizationResponse, err error) {
				expectedOrg := pb.OrganizationRecord{
					Id:        1,
					Name:      "TestOrg",
					Company:   "TestCompany",
					CreatedAt: timestamppb.New(time.Now()),
					UpdatedAt: timestamppb.New(time.Now()),
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, expectedOrg.Id, res.Organization.Id)
				assert.Equal(t, expectedOrg.Name, res.Organization.Name)
				assert.Equal(t, expectedOrg.Company, res.Organization.Company)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "NonExisting",
			req:  &pb.GetOrganizationRequest{OrganizationId: 5},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetOrganization(gomock.Any(), gomock.Any()).
					Return(db.Organization{}, nil).
					Times(1)
				store.EXPECT().ListRoles(gomock.Any(), gomock.Any())
				store.EXPECT().ListGroupsByOrganizationID(gomock.Any(), gomock.Any())
				store.EXPECT().ListUsersByOrganization(gomock.Any(), gomock.Any())
			},
			checkResponse: func(t *testing.T, res *pb.GetOrganizationResponse, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int32(0), res.Organization.Id)
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

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.GetOrganization(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetOrganizationByNameDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetOrganizationByNameRequest{Name: "TestOrg"}

	expectedOrg := db.Organization{
		ID:        1,
		Name:      "TestOrg",
		Company:   "TestCompany",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	mockStore.EXPECT().GetOrganizationByName(ctx, gomock.Any()).
		Return(expectedOrg, nil)
	mockStore.EXPECT().ListGroupsByOrganizationID(ctx, gomock.Any())
	mockStore.EXPECT().ListRoles(ctx, gomock.Any())
	mockStore.EXPECT().ListUsersByOrganization(ctx, gomock.Any())

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetOrganizationByName(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedOrg.ID, response.Organization.Id)
	assert.Equal(t, expectedOrg.Name, response.Organization.Name)
	assert.Equal(t, expectedOrg.Company, response.Organization.Company)
	expectedCreatedAt := expectedOrg.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Organization.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedOrg.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Organization.UpdatedAt.AsTime().In(time.UTC))
}

func TestGetNonExistingOrganizationByNameDBMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.GetOrganizationByNameRequest{Name: "Test"}
	// Create a new context and set the claims value
	ctx := context.WithValue(context.Background(), auth.TokenInfoKey, auth.UserClaims{
		UserId:         1,
		OrganizationId: 1,
		GroupIds:       []int32{1},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, GroupID: 0, OrganizationID: 1}},
	})

	mockStore.EXPECT().GetOrganizationByName(ctx, gomock.Any()).
		Return(db.Organization{}, nil)
	mockStore.EXPECT().ListGroupsByOrganizationID(ctx, gomock.Any())
	mockStore.EXPECT().ListRoles(ctx, gomock.Any())
	mockStore.EXPECT().ListUsersByOrganization(ctx, gomock.Any())

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetOrganizationByName(ctx, request)

	assert.NoError(t, err)
	assert.Equal(t, int32(0), response.Organization.Id)
}

func TestGetOrganizationByName_gRPC(t *testing.T) {
	testCases := []struct {
		name               string
		req                *pb.GetOrganizationByNameRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetOrganizationByNameResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetOrganizationByNameRequest{Name: "TestOrg"},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetOrganizationByName(gomock.Any(), gomock.Any()).
					Return(db.Organization{
						ID:        1,
						Name:      "TestOrg",
						Company:   "TestCompany",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil).
					Times(1)
				store.EXPECT().ListGroupsByOrganizationID(gomock.Any(), gomock.Any())
				store.EXPECT().ListRoles(gomock.Any(), gomock.Any())
				store.EXPECT().ListUsersByOrganization(gomock.Any(), gomock.Any())

			},
			checkResponse: func(t *testing.T, res *pb.GetOrganizationByNameResponse, err error) {
				expectedOrg := pb.OrganizationRecord{
					Id:        1,
					Name:      "TestOrg",
					Company:   "TestCompany",
					CreatedAt: timestamppb.New(time.Now()),
					UpdatedAt: timestamppb.New(time.Now()),
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, expectedOrg.Id, res.Organization.Id)
				assert.Equal(t, expectedOrg.Name, res.Organization.Name)
				assert.Equal(t, expectedOrg.Company, res.Organization.Company)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "NonExisting",
			req:  &pb.GetOrganizationByNameRequest{Name: "test"},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetOrganizationByName(gomock.Any(), gomock.Any()).
					Return(db.Organization{}, nil).
					Times(1)
				store.EXPECT().ListGroupsByOrganizationID(gomock.Any(), gomock.Any())
				store.EXPECT().ListRoles(gomock.Any(), gomock.Any())
				store.EXPECT().ListUsersByOrganization(gomock.Any(), gomock.Any())
			},
			checkResponse: func(t *testing.T, res *pb.GetOrganizationByNameResponse, err error) {
				assert.NoError(t, err)
				assert.Equal(t, int32(0), res.Organization.Id)
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

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.GetOrganizationByName(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestDeleteOrganizationDBMock(t *testing.T) {
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

	request := &pb.DeleteOrganizationRequest{Id: 1}

	expectedOrg := db.Organization{
		ID:        1,
		Name:      "test",
		Company:   "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockStore.EXPECT().GetOrganization(ctx, gomock.Any()).
		Return(expectedOrg, nil)
	mockStore.EXPECT().
		ListGroupsByOrganizationID(ctx, gomock.Any()).
		Return([]db.Group{}, nil)
	mockStore.EXPECT().
		DeleteOrganization(ctx, gomock.Any()).
		Return(nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.DeleteOrganization(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestDeleteOrganization_gRPC(t *testing.T) {
	force := true

	testCases := []struct {
		name               string
		req                *pb.DeleteOrganizationRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.DeleteOrganizationResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.DeleteOrganizationRequest{
				Id:    1,
				Force: &force,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetOrganization(gomock.Any(), gomock.Any()).Return(db.Organization{}, nil).Times(1)
				store.EXPECT().
					DeleteOrganization(gomock.Any(), gomock.Any()).Return(nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.DeleteOrganizationResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, &pb.DeleteOrganizationResponse{}, res)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req: &pb.DeleteOrganizationRequest{
				Id: 0,
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(t *testing.T, res *pb.DeleteOrganizationResponse, err error) {
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

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server, err := NewServer(mockStore, &config.Config{})
			require.NoError(t, err, "failed to create test server")

			resp, err := server.DeleteOrganization(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
