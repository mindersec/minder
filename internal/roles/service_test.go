//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package roles

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/stacklok/minder/internal/auth"
	mockauth "github.com/stacklok/minder/internal/auth/mock"
	"github.com/stacklok/minder/internal/authz"
	"github.com/stacklok/minder/internal/authz/mock"
	"github.com/stacklok/minder/internal/db"
	dbf "github.com/stacklok/minder/internal/db/fixtures"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestCreateRoleAssignment(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name                    string
		dBSetup                 dbf.DBMockBuilder
		existingRoleAssignments []*minderv1.RoleAssignment
		expectedError           string
	}{
		{
			name: "error when user doesn't exist",
			dBSetup: dbf.NewDBMock(
				withGetUser(emptyUser, sql.ErrNoRows),
			),
			expectedError: "User not found",
		},
		{
			name: "error when role assignment already exists",
			dBSetup: dbf.NewDBMock(
				withGetUser(validUser, nil),
			),
			existingRoleAssignments: []*minderv1.RoleAssignment{
				{
					Subject: subject,
					Role:    string(authz.RoleViewer),
				},
			},
			expectedError: "role assignment for this user already exists, use update instead",
		},
		{
			name: "role assignment created successfully",
			dBSetup: dbf.NewDBMock(
				withGetUser(validUser, nil),
			),
			existingRoleAssignments: []*minderv1.RoleAssignment{},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			var store db.Store
			if scenario.dBSetup != nil {
				store = scenario.dBSetup(ctrl)
			}

			authzClient := &mock.SimpleClient{
				Allowed: []uuid.UUID{project},
				Assignments: map[uuid.UUID][]*minderv1.RoleAssignment{
					project: scenario.existingRoleAssignments,
				},
			}

			idClient := mockauth.NewMockResolver(ctrl)
			idClient.EXPECT().Resolve(ctx, subject).Return(&auth.Identity{
				UserID: subject,
			}, nil)

			service := NewRoleService()
			_, err := service.CreateRoleAssignment(ctx, store, authzClient, idClient, project, subject, userRole)

			if scenario.expectedError != "" {
				require.ErrorContains(t, err, scenario.expectedError)
				return
			}
			require.NoError(t, err)

			// Verify the assignment is created
			require.Equal(t, 1, len(authzClient.Assignments[project]))
			require.Equal(t, userRole.String(), authzClient.Assignments[project][0].Role)
			require.Equal(t, subject, authzClient.Assignments[project][0].Subject)
		})
	}
}

func TestUpdateRoleAssignment(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name          string
		dBSetup       dbf.DBMockBuilder
		expectedError string
	}{
		{
			name: "error when user doesn't exist",
			dBSetup: dbf.NewDBMock(
				withGetUser(emptyUser, sql.ErrNoRows),
			),
			expectedError: "User not found",
		},
		{
			name: "role assignment updated successfully",
			dBSetup: dbf.NewDBMock(
				withGetUser(validUser, nil),
			),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			var store db.Store
			if scenario.dBSetup != nil {
				store = scenario.dBSetup(ctrl)
			}

			authzClient := &mock.SimpleClient{
				Allowed: []uuid.UUID{project},
				Assignments: map[uuid.UUID][]*minderv1.RoleAssignment{
					project: {
						{
							Subject: subject,
							Role:    string(authz.RoleViewer),
						},
					},
				},
			}

			idClient := mockauth.NewMockResolver(ctrl)
			idClient.EXPECT().Resolve(ctx, subject).Return(&auth.Identity{
				UserID: subject,
			}, nil)

			service := NewRoleService()
			_, err := service.UpdateRoleAssignment(ctx, store, authzClient, idClient, project, subject, userRole)

			if scenario.expectedError != "" {
				require.ErrorContains(t, err, scenario.expectedError)
				return
			}
			require.NoError(t, err)

			// Verify only the role is updated
			require.Equal(t, 1, len(authzClient.Assignments[project]))
			require.Equal(t, userRole.String(), authzClient.Assignments[project][0].Role)
			require.Equal(t, subject, authzClient.Assignments[project][0].Subject)
		})
	}
}

func TestRemoveRole(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name          string
		dBSetup       dbf.DBMockBuilder
		role          authz.Role
		expectedError string
		noAssignment  bool
	}{
		{
			name: "error when user doesn't exist",
			dBSetup: dbf.NewDBMock(
				withGetUser(emptyUser, sql.ErrNoRows),
			),
			expectedError: "User not found",
		},
		{
			name: "error when deleting last project admin",
			dBSetup: dbf.NewDBMock(
				withGetUser(validUser, nil),
			),
			role:          authz.RoleAdmin,
			expectedError: "cannot remove the last admin from the project",
		},
		{
			name: "role deleted successfully",
			role: authz.RoleViewer,
			dBSetup: dbf.NewDBMock(
				withGetUser(validUser, nil),
			),
		},
		{
			name: "error when role assignment doesn't exist",
			role: authz.RoleEditor,
			dBSetup: dbf.NewDBMock(
				withGetUser(validUser, nil),
			),
			noAssignment:  true,
			expectedError: "role assignment for this user does not exist",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ctx := context.Background()

			var store db.Store
			if scenario.dBSetup != nil {
				store = scenario.dBSetup(ctrl)
			}

			idClient := mockauth.NewMockResolver(ctrl)
			idClient.EXPECT().Resolve(ctx, subject).Return(&auth.Identity{
				UserID: subject,
			}, nil)

			authzClient := &mock.SimpleClient{
				Assignments: map[uuid.UUID][]*minderv1.RoleAssignment{
					project: {
						{
							Subject: subject,
							Role:    scenario.role.String(),
						},
					},
				},
			}

			if scenario.noAssignment {
				authzClient.Assignments[project] = []*minderv1.RoleAssignment{}
			}

			service := NewRoleService()
			_, err := service.RemoveRoleAssignment(ctx, store, authzClient, idClient, project, subject, scenario.role)

			if scenario.expectedError != "" {
				require.ErrorContains(t, err, scenario.expectedError)
				return
			}
			require.NoError(t, err)

			// verify the only role is removed
			require.Equal(t, 0, len(authzClient.Assignments[project]))
		})
	}
}

var (
	project  = uuid.New()
	subject  = "subject"
	userRole = authz.RoleAdmin

	emptyUser = db.User{}
	validUser = db.User{
		ID: 1,
	}
)

func withGetUser(result db.User, err error) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			GetUserBySubject(gomock.Any(), gomock.Any()).
			Return(result, err)
	}
}
