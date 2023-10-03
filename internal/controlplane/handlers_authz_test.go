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
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	mockdb "github.com/stacklok/mediator/database/mock"
	"github.com/stacklok/mediator/internal/auth"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

func TestIsSuperadminAuthorized(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orgID := uuid.New()
	projectID := uuid.New()

	request := &pb.GetProjectByIdRequest{ProjectId: projectID.String()}
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projectID},
		IsStaff:        true, // TODO: remove this
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: true, ProjectID: &projectID, OrganizationID: orgID}},
	})

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().GetProjectByID(ctx, gomock.Any())
	mockStore.EXPECT().ListRolesByProjectID(ctx, gomock.Any())
	mockStore.EXPECT().ListUsersByProject(ctx, gomock.Any())

	server := &Server{
		store: mockStore,
	}

	response, err := server.GetProjectById(ctx, request)

	assert.NoError(t, err)
	assert.NotNil(t, response)
}

func TestIsNonadminAuthorized(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orgID := uuid.New()
	projectID := uuid.New()

	request := &pb.CreateRoleByOrganizationRequest{OrganizationId: orgID.String(), Name: "test"}
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         1,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projectID},
		Roles: []auth.RoleInfo{
			{RoleID: 1, IsAdmin: false, ProjectID: &projectID, OrganizationID: orgID}},
	})

	rpcOpts, err := optionsForMethod(&grpc.UnaryServerInfo{FullMethod: "/mediator.v1.RoleService/CreateRoleByOrganization"})
	if err != nil {
		t.Fatalf("Unable to get rpc options: %v", err)
	}
	ctx = withRpcOptions(ctx, rpcOpts)

	mockStore := mockdb.NewMockStore(ctrl)
	server := &Server{
		store: mockStore,
	}
	mockStore.EXPECT().CreateRole(ctx, gomock.Any()).Times(0)

	_, err = server.CreateRoleByOrganization(ctx, request)

	t.Logf("Got error: %v", err)

	if err == nil {
		t.Error("Expected error when user is not authorized, but got nil")
	} else {
		t.Logf("Successfully received error when user is not authorized: %v", err)
	}
}

func TestByResourceUnauthorized(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	orgID := uuid.New()
	projectID1 := uuid.New()
	projectID2 := uuid.New()

	request := &pb.GetRoleByIdRequest{Id: 1}
	// Create a new context and set the claims value
	ctx := auth.WithPermissionsContext(context.Background(), auth.UserPermissions{
		UserId:         2,
		OrganizationId: orgID,
		ProjectIds:     []uuid.UUID{projectID1},
		Roles: []auth.RoleInfo{
			{RoleID: 2, IsAdmin: true, ProjectID: &projectID2, OrganizationID: orgID}},
	})

	rpcOpts, err := optionsForMethod(&grpc.UnaryServerInfo{FullMethod: "/mediator.v1.RoleService/GetRoleById"})
	if err != nil {
		t.Fatalf("Unable to get rpc options: %v", err)
	}
	ctx = withRpcOptions(ctx, rpcOpts)

	mockStore := mockdb.NewMockStore(ctrl)
	server := &Server{
		store: mockStore,
	}
	mockStore.EXPECT().GetRoleByID(ctx, gomock.Any()).Times(1)

	_, err = server.GetRoleById(ctx, request)

	if err == nil {
		t.Error("Expected error when user is not authorized, but got nil")
	} else {
		t.Logf("Successfully received error when user is not authorized: %v", err)
	}
}
