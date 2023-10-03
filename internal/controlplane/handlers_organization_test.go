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
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	mockdb "github.com/stacklok/mediator/database/mock"
	"github.com/stacklok/mediator/internal/auth"
	"github.com/stacklok/mediator/internal/db"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

func TestCreateOrganizationDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	request := &pb.CreateOrganizationRequest{
		Name:    "TestOrg",
		Company: "TestCompany",
	}

	projID := uuid.New()
	expectedOrg := db.Project{
		ID:             uuid.New(),
		Name:           "TestOrg",
		IsOrganization: true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: expectedOrg.ID,
		ProjectIds:     []uuid.UUID{projID},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projID, OrganizationID: expectedOrg.ID}},
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
	assert.Equal(t, expectedOrg.ID.String(), response.Id)
	assert.Equal(t, expectedOrg.Name, response.Name)
	// assert.Equal(t, expectedOrg.Company, response.Company)
	expectedCreatedAt := expectedOrg.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedOrg.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.UpdatedAt.AsTime().In(time.UTC))
}

func TestCreateOrganization_gRPC(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	projectID := uuid.New()

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
				meta, err := json.Marshal(&OrgMeta{
					Company: "TestCompany",
				})
				assert.NoError(t, err, "unexpected error marshalling metadata")
				store.EXPECT().
					CreateOrganization(gomock.Any(), gomock.Any()).
					Return(db.Project{
						ID:        orgID,
						Name:      "TestOrg",
						Metadata:  meta,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil).
					Times(1)
				store.EXPECT().Commit(gomock.Any())
				store.EXPECT().Rollback(gomock.Any())
			},
			checkResponse: func(t *testing.T, res *pb.CreateOrganizationResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, orgID.String(), res.Id)
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
				t.Helper()

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
					Return(db.Project{}, errors.New("store error")).
					Times(1)
				store.EXPECT().Rollback(gomock.Any())
			},
			checkResponse: func(t *testing.T, res *pb.CreateOrganizationResponse, err error) {
				t.Helper()

				// Assert the expected behavior when there's a store error
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.Internal,
		},
	}
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projectID},
		IsStaff:        true, // TODO: remove this
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

			server := newDefaultServer(t, mockStore)

			resp, err := server.CreateOrganization(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetOrganizationsDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orgID := uuid.New()
	projID := uuid.New()

	mockStore := mockdb.NewMockStore(ctrl)
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projID},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projID, OrganizationID: orgID}},
	})

	request := &pb.GetOrganizationsRequest{}

	orgMeta1 := &OrgMeta{
		Company: "TestCompany",
	}

	orgMeta2 := &OrgMeta{
		Company: "TestCompany1",
	}

	marshalledMeta1, err := json.Marshal(orgMeta1)
	assert.NoError(t, err, "unexpected error marshalling metadata")

	marshalledMeta2, err := json.Marshal(orgMeta2)
	assert.NoError(t, err, "unexpected error marshalling metadata")

	expectedOrgs := []db.Project{
		{
			ID:        uuid.New(),
			Name:      "TestOrg",
			Metadata:  marshalledMeta1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Name:      "TestOrg1",
			Metadata:  marshalledMeta2,
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
	assert.Equal(t, expectedOrgs[0].ID.String(), response.Organizations[0].Id)
	assert.Equal(t, expectedOrgs[0].Name, response.Organizations[0].Name)
	assert.Contains(t, string(expectedOrgs[0].Metadata), response.Organizations[0].Company)
	expectedCreatedAt := expectedOrgs[0].CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Organizations[0].CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedOrgs[0].UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Organizations[0].UpdatedAt.AsTime().In(time.UTC))
}

func TestGetOrganizations_gRPC(t *testing.T) {
	t.Parallel()

	orgID1 := uuid.New()
	orgID2 := uuid.New()
	projID := uuid.New()

	orgmeta1 := &OrgMeta{
		Company: "TestCompany",
	}
	orgmeta2 := &OrgMeta{
		Company: "TestCompany1",
	}

	marshalledMeta1, err := json.Marshal(orgmeta1)
	assert.NoError(t, err, "unexpected error marshalling metadata")
	marshalledMeta2, err := json.Marshal(orgmeta2)
	assert.NoError(t, err, "unexpected error marshalling metadata")

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
					Return([]db.Project{
						{
							ID:        orgID1,
							Name:      "TestOrg",
							Metadata:  marshalledMeta1,
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						{
							ID:        orgID2,
							Name:      "TestOrg1",
							Metadata:  marshalledMeta2,
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetOrganizationsResponse, err error) {
				t.Helper()

				expectedOrgs := []*pb.OrganizationRecord{
					{
						Id:        orgID1.String(),
						Name:      "TestOrg",
						Company:   "TestCompany",
						CreatedAt: timestamppb.New(time.Now()),
						UpdatedAt: timestamppb.New(time.Now()),
					},
					{
						Id:        orgID2.String(),
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
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID1,
		ProjectIds:     []uuid.UUID{projID},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projID, OrganizationID: orgID1}},
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := newDefaultServer(t, mockStore)

			resp, err := server.GetOrganizations(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetOrganizationDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	orgID := uuid.New()
	projID := uuid.New()
	orgmeta := &OrgMeta{
		Company: "TestCompany",
	}

	marshalledMeta, err := json.Marshal(orgmeta)
	assert.NoError(t, err, "unexpected error marshalling metadata")

	request := &pb.GetOrganizationRequest{OrganizationId: orgID.String()}

	expectedOrg := db.Project{
		ID:        orgID,
		Name:      "TestOrg",
		Metadata:  marshalledMeta,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: expectedOrg.ID,
		ProjectIds:     []uuid.UUID{projID},
		IsStaff:        true, // TODO: remove this
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projID, OrganizationID: expectedOrg.ID}},
	})
	mockStore.EXPECT().GetOrganization(ctx, gomock.Any()).
		Return(expectedOrg, nil)
	mockStore.EXPECT().GetChildrenProjects(ctx, gomock.Any())
	mockStore.EXPECT().ListRoles(ctx, gomock.Any())
	mockStore.EXPECT().ListUsersByOrganization(ctx, gomock.Any())

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetOrganization(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedOrg.ID.String(), response.Organization.Id)
	assert.Equal(t, expectedOrg.Name, response.Organization.Name)
	assert.Contains(t, string(expectedOrg.Metadata), response.Organization.Company)
	expectedCreatedAt := expectedOrg.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Organization.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedOrg.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Organization.UpdatedAt.AsTime().In(time.UTC))
}

func TestGetNonExistingOrganizationDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orgID := uuid.New()
	projectID := uuid.New()

	mockStore := mockdb.NewMockStore(ctrl)
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projectID},
		IsStaff:        true, // TODO: remove this
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projectID, OrganizationID: orgID}},
	})

	unexistentOrgID := uuid.New()

	request := &pb.GetOrganizationRequest{OrganizationId: unexistentOrgID.String()}

	mockStore.EXPECT().GetOrganization(ctx, gomock.Any()).
		Return(db.Project{}, sql.ErrNoRows)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetOrganization(ctx, request)

	assert.Error(t, err, "expected error when organization does not exist")
	assert.Nil(t, response)
}

func TestGetOrganization_gRPC(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	orgID := uuid.New()
	orgmeta := &OrgMeta{
		Company: "TestCompany",
	}

	marshalledmeta, err := json.Marshal(orgmeta)
	assert.NoError(t, err, "unexpected error marshalling metadata")

	testCases := []struct {
		name               string
		req                *pb.GetOrganizationRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetOrganizationResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetOrganizationRequest{OrganizationId: orgID.String()},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetOrganization(gomock.Any(), gomock.Any()).
					Return(db.Project{
						ID:        orgID,
						Name:      "TestOrg",
						Metadata:  marshalledmeta,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil).
					Times(1)
				store.EXPECT().ListRoles(gomock.Any(), gomock.Any())
				store.EXPECT().GetChildrenProjects(gomock.Any(), orgID)
				store.EXPECT().ListUsersByOrganization(gomock.Any(), gomock.Any())

			},
			checkResponse: func(t *testing.T, res *pb.GetOrganizationResponse, err error) {
				t.Helper()

				expectedOrg := pb.OrganizationRecord{
					Id:        orgID.String(),
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
			req:  &pb.GetOrganizationRequest{OrganizationId: uuid.NewString()},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetOrganization(gomock.Any(), gomock.Any()).
					Return(db.Project{}, sql.ErrNoRows).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetOrganizationResponse, err error) {
				t.Helper()

				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.OK,
		},
	}

	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projectID},
		IsStaff:        true, // TODO: remove this
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

			server := newDefaultServer(t, mockStore)

			resp, err := server.GetOrganization(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetOrganizationByNameDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	projectID := uuid.New()
	orgID := uuid.New()

	orgmeta := &OrgMeta{
		Company: "TestCompany",
	}

	marshalledMeta, err := json.Marshal(orgmeta)
	assert.NoError(t, err, "unexpected error marshalling metadata")

	request := &pb.GetOrganizationByNameRequest{Name: "TestOrg"}

	expectedOrg := db.Project{
		ID:        orgID,
		Name:      "TestOrg",
		Metadata:  marshalledMeta,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projectID},
		IsStaff:        true, // TODO: remove this
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projectID, OrganizationID: orgID}},
	})

	mockStore.EXPECT().GetOrganizationByName(ctx, gomock.Any()).
		Return(expectedOrg, nil)
	mockStore.EXPECT().GetChildrenProjects(ctx, gomock.Any())
	mockStore.EXPECT().ListRoles(ctx, gomock.Any())
	mockStore.EXPECT().ListUsersByOrganization(ctx, gomock.Any())

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetOrganizationByName(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedOrg.ID.String(), response.Organization.Id)
	assert.Equal(t, expectedOrg.Name, response.Organization.Name)
	assert.Contains(t, string(expectedOrg.Metadata), response.Organization.Company)
	expectedCreatedAt := expectedOrg.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Organization.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedOrg.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Organization.UpdatedAt.AsTime().In(time.UTC))
}

func TestGetNonExistingOrganizationByNameDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	orgID := uuid.New()
	projID := uuid.New()

	request := &pb.GetOrganizationByNameRequest{Name: "Test"}
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projID},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projID, OrganizationID: orgID}},
	})

	mockStore.EXPECT().GetOrganizationByName(ctx, gomock.Any()).
		Return(db.Project{}, sql.ErrNoRows)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetOrganizationByName(ctx, request)

	assert.Error(t, err)
	assert.Nil(t, response)
}

func TestGetOrganizationByName_gRPC(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	projectID := uuid.New()

	orgmeta := &OrgMeta{
		Company: "TestCompany",
	}

	marshalledMeta, err := json.Marshal(orgmeta)
	assert.NoError(t, err, "unexpected error marshalling metadata")

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
					Return(db.Project{
						ID:        orgID,
						Name:      "TestOrg",
						Metadata:  marshalledMeta,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil).
					Times(1)
				store.EXPECT().GetChildrenProjects(gomock.Any(), gomock.Any())
				store.EXPECT().ListRoles(gomock.Any(), gomock.Any())
				store.EXPECT().ListUsersByOrganization(gomock.Any(), gomock.Any())

			},
			checkResponse: func(t *testing.T, res *pb.GetOrganizationByNameResponse, err error) {
				t.Helper()

				expectedOrg := pb.OrganizationRecord{
					Id:        orgID.String(),
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
					Return(db.Project{}, sql.ErrNoRows).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetOrganizationByNameResponse, err error) {
				t.Helper()

				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.OK,
		},
	}
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projectID},
		IsStaff:        true,
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

			server := newDefaultServer(t, mockStore)

			resp, err := server.GetOrganizationByName(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestDeleteOrganizationDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orgID := uuid.New()
	projID := uuid.New()

	orgmeta := &OrgMeta{
		Company: "test",
	}

	marshalledMeta, err := json.Marshal(orgmeta)
	assert.NoError(t, err, "unexpected error marshalling metadata")

	mockStore := mockdb.NewMockStore(ctrl)
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projID},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projID, OrganizationID: orgID}},
	})

	request := &pb.DeleteOrganizationRequest{Id: orgID.String()}

	expectedOrg := db.Project{
		ID:        orgID,
		Name:      "test",
		Metadata:  marshalledMeta,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockStore.EXPECT().GetOrganization(ctx, gomock.Any()).
		Return(expectedOrg, nil)
	mockStore.EXPECT().
		GetChildrenProjects(ctx, gomock.Any()).
		Return([]db.GetChildrenProjectsRow{
			{
				ID: orgID,
			},
		}, nil)
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
	t.Parallel()

	force := true

	orgID := uuid.New()
	projID := uuid.New()

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
				Id:    orgID.String(),
				Force: &force,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetOrganization(gomock.Any(), gomock.Any()).Return(db.Project{}, nil).Times(1)
				store.EXPECT().
					DeleteOrganization(gomock.Any(), gomock.Any()).Return(nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.DeleteOrganizationResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, &pb.DeleteOrganizationResponse{}, res)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req: &pb.DeleteOrganizationRequest{
				Id: "",
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(t *testing.T, res *pb.DeleteOrganizationResponse, err error) {
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
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projID},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projID, OrganizationID: orgID}},
	})

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := newDefaultServer(t, mockStore)

			resp, err := server.DeleteOrganization(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
