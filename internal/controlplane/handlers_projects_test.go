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

func TestCreateProjectDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	orgID := uuid.New()
	projID := uuid.New()

	request := &pb.CreateProjectRequest{
		OrganizationId: orgID.String(),
		Name:           "TestProject",
		Description:    "TestDescription",
		IsProtected:    nil,
	}

	expectedProject := db.Project{
		ID: projID,
		ParentID: uuid.NullUUID{
			UUID:  orgID,
			Valid: true,
		},
		Name: "TestProject",
	}

	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projID},
		IsStaff:        true, // TODO: remove this
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projID, OrganizationID: orgID}},
	})

	mockStore.EXPECT().
		CreateProject(ctx, gomock.Any()).
		Return(expectedProject, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.CreateProject(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedProject.ID.String(), response.ProjectId)
	assert.Equal(t, expectedProject.Name, response.Name)
	assert.Equal(t, expectedProject.ParentID.UUID.String(), response.OrganizationId)
}

func TestCreateProject_gRPC(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	projID := uuid.New()

	projmeta := &ProjectMeta{
		Description: "TestDescription",
		IsProtected: false,
	}

	projMetaJSON, err := json.Marshal(projmeta)
	assert.NoError(t, err, "failed to marshal project metadata")

	testCases := []struct {
		name               string
		req                *pb.CreateProjectRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.CreateProjectResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.CreateProjectRequest{
				OrganizationId: orgID.String(),
				Name:           "TestProject",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateProject(gomock.Any(), gomock.Any()).
					Return(db.Project{
						ID: projID,
						ParentID: uuid.NullUUID{
							UUID:  orgID,
							Valid: true,
						},
						Name:      "TestProject",
						Metadata:  projMetaJSON,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.CreateProjectResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, projID.String(), res.ProjectId)
				assert.Equal(t, orgID.String(), res.OrganizationId)
				assert.Equal(t, "TestProject", res.Name)
				assert.Equal(t, false, res.IsProtected)
				assert.NotNil(t, res.CreatedAt)
				assert.NotNil(t, res.UpdatedAt)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req: &pb.CreateProjectRequest{
				Name: "",
			},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(t *testing.T, res *pb.CreateProjectResponse, err error) {
				t.Helper()

				// Assert the expected behavior when the request is empty
				assert.Error(t, err)
				assert.Nil(t, res)
			},
			expectedStatusCode: codes.InvalidArgument,
		},
		{
			name: "StoreError",
			req: &pb.CreateProjectRequest{
				OrganizationId: orgID.String(),
				Name:           "TestProject",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateProject(gomock.Any(), gomock.Any()).
					Return(db.Project{}, errors.New("store error")).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.CreateProjectResponse, err error) {
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
		ProjectIds:     []uuid.UUID{projID},
		IsStaff:        true,
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

			resp, err := server.CreateProject(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestDeleteProjectDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	orgID := uuid.New()
	projID := uuid.New()

	request := &pb.DeleteProjectRequest{Id: projID.String()}

	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projID},
		IsStaff:        true, // TODO: remove this
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projID, OrganizationID: orgID}},
	})

	mockStore.EXPECT().
		GetProjectByID(gomock.Any(), gomock.Any())
	mockStore.EXPECT().
		ListRolesByProjectID(ctx, gomock.Any()).
		Return([]db.Role{}, nil)
	mockStore.EXPECT().
		DeleteProject(ctx, gomock.Any()).
		Return(nil, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.DeleteProject(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestDeleteProject_gRPC(t *testing.T) {
	t.Parallel()

	force := true

	orgID := uuid.New()
	projID := uuid.New()

	testCases := []struct {
		name               string
		req                *pb.DeleteProjectRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.DeleteProjectResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req: &pb.DeleteProjectRequest{
				Id:    projID.String(),
				Force: &force,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetProjectByID(gomock.Any(), gomock.Any()).Return(db.Project{}, nil)
				store.EXPECT().
					DeleteProject(gomock.Any(), gomock.Any()).Return(nil, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.DeleteProjectResponse, err error) {
				t.Helper()

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, &pb.DeleteProjectResponse{}, res)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "EmptyRequest",
			req:  &pb.DeleteProjectRequest{},
			buildStubs: func(store *mockdb.MockStore) {
			},
			checkResponse: func(t *testing.T, res *pb.DeleteProjectResponse, err error) {
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
		IsStaff:        true, // TODO: remove this
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

			resp, err := server.DeleteProject(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetProjectsDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	orgID := uuid.New()
	projID := uuid.New()
	projID2 := uuid.New()

	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projID},
		IsStaff:        true, // TODO: remove this
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projID, OrganizationID: orgID}},
	})

	request := &pb.GetProjectsRequest{OrganizationId: orgID.String()}

	expectedProjects := []db.GetChildrenProjectsRow{
		{
			ID:   orgID,
			Name: "org",
		},
		{
			ID: projID,
			ParentID: uuid.NullUUID{
				UUID:  orgID,
				Valid: true,
			},
			Name:      "test",
			Metadata:  []byte(`{"is_protected": false}`),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID: projID2,
			ParentID: uuid.NullUUID{
				UUID:  orgID,
				Valid: true,
			},
			Name:      "test1",
			Metadata:  []byte(`{"is_protected": true}`),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	mockStore.EXPECT().GetChildrenProjects(ctx, gomock.Any()).
		Return(expectedProjects, nil)

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetProjects(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, len(expectedProjects), db.CalculateProjectHierarchyOffset(len(response.Projects)))
	assert.Equal(t, expectedProjects[1].ID.String(), response.Projects[0].ProjectId)
	assert.Equal(t, expectedProjects[1].ParentID.UUID.String(), response.Projects[0].OrganizationId)
	assert.Equal(t, expectedProjects[1].Name, response.Projects[0].Name)

	expectedCreatedAt := expectedProjects[1].CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Projects[0].CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedProjects[1].UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Projects[0].UpdatedAt.AsTime().In(time.UTC))
}

func TestGetProjects_gRPC(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	projID1 := uuid.New()
	projID2 := uuid.New()

	testCases := []struct {
		name               string
		req                *pb.GetProjectsRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetProjectsResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetProjectsRequest{OrganizationId: orgID.String()},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetChildrenProjects(gomock.Any(), gomock.Any()).
					Return([]db.GetChildrenProjectsRow{
						{
							ID:   orgID,
							Name: "org",
						},
						{
							ID: projID1,
							ParentID: uuid.NullUUID{
								UUID:  orgID,
								Valid: true,
							},
							Name:      "test",
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
						{
							ID: projID2,
							ParentID: uuid.NullUUID{
								UUID:  orgID,
								Valid: true,
							},
							Name:      "test1",
							CreatedAt: time.Now(),
							UpdatedAt: time.Now(),
						},
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetProjectsResponse, err error) {
				t.Helper()

				expectedProjects := []*pb.ProjectRecord{
					{
						ProjectId:      projID1.String(),
						OrganizationId: orgID.String(),
						Name:           "test",
						CreatedAt:      timestamppb.New(time.Now()),
						UpdatedAt:      timestamppb.New(time.Now()),
					},
					{
						ProjectId:      projID2.String(),
						OrganizationId: orgID.String(),
						Name:           "test1",
						CreatedAt:      timestamppb.New(time.Now()),
						UpdatedAt:      timestamppb.New(time.Now()),
					},
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, len(expectedProjects), len(res.Projects))
				assert.Equal(t, expectedProjects[0].OrganizationId, res.Projects[0].OrganizationId)
				assert.Equal(t, expectedProjects[0].ProjectId, res.Projects[0].ProjectId)
				assert.Equal(t, expectedProjects[0].Name, res.Projects[0].Name)
			},
			expectedStatusCode: codes.OK,
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create a new context and set the claims value
			ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
				UserId:         1,
				OrganizationId: orgID,
				ProjectIds:     []uuid.UUID{projID1},
				IsStaff:        true, // TODO: remove this
				Roles: []auth.RoleInfo{
					{RoleID: 1, IsAdmin: true, ProjectID: &projID1, OrganizationID: orgID}},
			})

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := newDefaultServer(t, mockStore)

			resp, err := server.GetProjects(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}

func TestGetProjectDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	orgID := uuid.New()
	projID := uuid.New()

	request := &pb.GetProjectByIdRequest{ProjectId: projID.String()}
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projID},
		IsStaff:        true, // TODO: remove this
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projID, OrganizationID: orgID}},
	})

	expectedProject := db.Project{
		ID:        projID,
		ParentID:  uuid.NullUUID{UUID: orgID, Valid: true},
		Name:      "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	mockStore.EXPECT().GetProjectByID(ctx, gomock.Any()).
		Return(expectedProject, nil)
	mockStore.EXPECT().ListRolesByProjectID(ctx, gomock.Any())
	mockStore.EXPECT().ListUsersByProject(ctx, gomock.Any())

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetProjectById(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, expectedProject.ID.String(), response.Project.ProjectId)
	assert.Equal(t, expectedProject.ParentID.UUID.String(), response.Project.OrganizationId)
	assert.Equal(t, expectedProject.Name, response.Project.Name)
	expectedCreatedAt := expectedProject.CreatedAt.In(time.UTC)
	assert.Equal(t, expectedCreatedAt, response.Project.CreatedAt.AsTime().In(time.UTC))
	expectedUpdatedAt := expectedProject.UpdatedAt.In(time.UTC)
	assert.Equal(t, expectedUpdatedAt, response.Project.UpdatedAt.AsTime().In(time.UTC))
}

func TestGetNonExistingProjectDBMock(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)

	orgID := uuid.New()
	projID := uuid.New()

	request := &pb.GetProjectByIdRequest{ProjectId: uuid.NewString()}
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projID},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projID, OrganizationID: orgID}},
	})

	mockStore.EXPECT().GetProjectByID(ctx, gomock.Any()).
		Return(db.Project{}, sql.ErrNoRows)

	server := &Server{
		store: mockStore,
	}

	_, err := server.GetProjectById(ctx, request)

	assert.Error(t, err)
}

func TestGetProject_gRPC(t *testing.T) {
	t.Parallel()

	orgID := uuid.New()
	projID := uuid.New()

	testCases := []struct {
		name               string
		req                *pb.GetProjectByIdRequest
		buildStubs         func(store *mockdb.MockStore)
		checkResponse      func(t *testing.T, res *pb.GetProjectByIdResponse, err error)
		expectedStatusCode codes.Code
	}{
		{
			name: "Success",
			req:  &pb.GetProjectByIdRequest{ProjectId: projID.String()},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListRolesByProjectID(gomock.Any(), gomock.Any())
				store.EXPECT().ListUsersByProject(gomock.Any(), gomock.Any())

				store.EXPECT().GetProjectByID(gomock.Any(), gomock.Any()).
					Return(db.Project{
						ID:        projID,
						ParentID:  uuid.NullUUID{UUID: orgID, Valid: true},
						Name:      "test",
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}, nil).
					Times(1)
			},
			checkResponse: func(t *testing.T, res *pb.GetProjectByIdResponse, err error) {
				t.Helper()

				expectedProject := pb.ProjectRecord{
					ProjectId:      projID.String(),
					OrganizationId: orgID.String(),
					Name:           "test",
					CreatedAt:      timestamppb.New(time.Now()),
					UpdatedAt:      timestamppb.New(time.Now()),
				}

				assert.NoError(t, err)
				assert.NotNil(t, res)
				assert.Equal(t, expectedProject.OrganizationId, res.Project.OrganizationId)
				assert.Equal(t, expectedProject.ProjectId, res.Project.ProjectId)
				assert.Equal(t, expectedProject.Name, res.Project.Name)
			},
			expectedStatusCode: codes.OK,
		},
		{
			name: "NonExisting",
			req:  &pb.GetProjectByIdRequest{ProjectId: uuid.NewString()},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetProjectByID(gomock.Any(), gomock.Any()).
					Return(db.Project{}, sql.ErrNoRows).
					Times(1)

			},
			checkResponse: func(t *testing.T, res *pb.GetProjectByIdResponse, err error) {
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
		ProjectIds:     []uuid.UUID{projID},
		IsStaff:        true, // TODO: remove this
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

			resp, err := server.GetProjectById(ctx, tc.req)
			tc.checkResponse(t, resp, err)
		})
	}
}
