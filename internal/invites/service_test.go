// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package invites

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/auth"
	authjwt "github.com/mindersec/minder/internal/auth/jwt"
	mockauth "github.com/mindersec/minder/internal/auth/mock"
	"github.com/mindersec/minder/internal/authz"
	"github.com/mindersec/minder/internal/db"
	dbf "github.com/mindersec/minder/internal/db/fixtures"
	"github.com/mindersec/minder/internal/email"
	mockevents "github.com/mindersec/minder/internal/events/mock"
	"github.com/mindersec/minder/internal/projects"
	minder "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/config/server"
)

func TestCreateInvite(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name           string
		dBSetup        dbf.DBMockBuilder
		publisherSetup func(t *testing.T, pub *mockevents.MockPublisher)
		expectedError  string
		expectedResult *minder.Invitation
	}{
		{
			name: "error when existing invites",
			dBSetup: dbf.NewDBMock(
				withGetUserBySubject(validUser),
				withExistingInvites(multipleInvites),
			),
			expectedError: "invitation for this email and project already exists, use update instead",
		},
		{
			name: "invite created and message sent successfully",
			dBSetup: dbf.NewDBMock(
				withGetUserBySubject(validUser),
				withExistingInvites(noInvites),
				withCreateInvite(userInvite, nil),
				withProject(),
			),
			publisherSetup: func(_ *testing.T, pub *mockevents.MockPublisher) {
				pub.EXPECT().Publish(email.TopicQueueInviteEmail, gomock.Any())
			},
			expectedResult: &minder.Invitation{
				Project: projectId.String(),
				Role:    userRole.String(),
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			user := openid.New()
			assert.NoError(t, user.Set("sub", userEmail))

			ctx := context.Background()
			ctx = authjwt.WithAuthTokenContext(ctx, user)

			idClient := mockauth.NewMockResolver(ctrl)
			idClient.EXPECT().Resolve(ctx, userSubject).Return(&auth.Identity{
				UserID: userSubject,
			}, nil).AnyTimes()

			publisher := mockevents.NewMockPublisher(ctrl)
			if scenario.publisherSetup != nil {
				scenario.publisherSetup(t, publisher)
			}

			emailConfig := server.EmailConfig{
				MinderURLBase: baseUrl,
			}

			service := NewInviteService()
			invite, err := service.CreateInvite(ctx, scenario.dBSetup(ctrl), idClient, publisher, emailConfig, projectId, userRole, userEmail)

			if scenario.expectedError != "" {
				require.ErrorContains(t, err, scenario.expectedError)
				return
			}
			require.NoError(t, err)

			if scenario.expectedResult != nil {
				require.Equal(t, scenario.expectedResult.Role, invite.Role)
				require.Equal(t, scenario.expectedResult.Project, invite.Project)
			}
		})
	}
}

func TestUpdateInvite(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name           string
		dBSetup        dbf.DBMockBuilder
		publisherSetup func(t *testing.T, pub *mockevents.MockPublisher)
		expectedError  string
		expectedResult *minder.Invitation
	}{
		{
			name: "error when no existing invites",
			dBSetup: dbf.NewDBMock(
				withGetUserBySubject(validUser),
				withExistingInvites(noInvites),
			),
			expectedError: "no invitations found for this email and project",
		},
		{
			name: "error when multiple existing invites",
			dBSetup: dbf.NewDBMock(
				withGetUserBySubject(validUser),
				withExistingInvites(multipleInvites),
			),
			expectedError: "multiple invitations found for this email and project",
		},
		{
			name: "no message sent when role is the same",
			dBSetup: dbf.NewDBMock(
				withGetUserBySubject(validUser),
				withExistingInvites(singleInviteWithSameRole),
				withInviteRoleUpdate(userInvite, nil),
				withProject(),
			),
		},
		{
			name: "invite updated and message sent successfully",
			dBSetup: dbf.NewDBMock(
				withGetUserBySubject(validUser),
				withExistingInvites(singleInviteWithDifferentRole),
				withInviteRoleUpdate(userInvite, nil),
				withProject(),
			),
			publisherSetup: func(_ *testing.T, pub *mockevents.MockPublisher) {
				pub.EXPECT().Publish(email.TopicQueueInviteEmail, gomock.Any())
			},
			expectedResult: &minder.Invitation{
				Project:   projectId.String(),
				Role:      userRole.String(),
				InviteUrl: fmt.Sprintf("%s/join/%s", baseUrl, inviteCode),
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			user := openid.New()
			assert.NoError(t, user.Set("sub", userEmail))

			ctx := context.Background()
			ctx = authjwt.WithAuthTokenContext(ctx, user)

			idClient := mockauth.NewMockResolver(ctrl)
			idClient.EXPECT().Resolve(ctx, userSubject).Return(&auth.Identity{
				UserID: userSubject,
			}, nil).AnyTimes()

			publisher := mockevents.NewMockPublisher(ctrl)
			if scenario.publisherSetup != nil {
				scenario.publisherSetup(t, publisher)
			}

			emailConfig := server.EmailConfig{
				MinderURLBase: baseUrl,
			}

			service := NewInviteService()
			invite, err := service.UpdateInvite(ctx, scenario.dBSetup(ctrl), idClient, publisher, emailConfig, projectId, userRole, userEmail)

			if scenario.expectedError != "" {
				require.ErrorContains(t, err, scenario.expectedError)
				return
			}
			require.NoError(t, err)

			if scenario.expectedResult != nil {
				require.Equal(t, scenario.expectedResult.Role, invite.Role)
				require.Equal(t, scenario.expectedResult.Project, invite.Project)
				require.Equal(t, scenario.expectedResult.InviteUrl, invite.InviteUrl)
			}
		})
	}
}

func TestRemoveInvite(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name           string
		dBSetup        dbf.DBMockBuilder
		expectedError  string
		expectedResult *minder.Invitation
	}{
		{
			name: "error when no existing invites",
			dBSetup: dbf.NewDBMock(
				withExistingInvites(noInvites),
			),
			expectedError: "no invitations found for this email and project",
		},
		{
			name: "error when no invite matches role",
			dBSetup: dbf.NewDBMock(
				withExistingInvites(singleInviteWithDifferentRole),
			),
			expectedError: "no invitation found for this role and email in the project",
		},
		{
			name: "no message sent when role is the same",
			dBSetup: dbf.NewDBMock(
				withExistingInvites(singleInviteWithSameRole),
				withDeleteInvite(userInvite, nil),
				withProject(),
				withGetUserByID(validUser),
			),
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			user := openid.New()
			assert.NoError(t, user.Set("sub", userEmail))

			ctx := context.Background()
			ctx = authjwt.WithAuthTokenContext(ctx, user)

			idClient := mockauth.NewMockResolver(ctrl)
			idClient.EXPECT().Resolve(ctx, userSubject).Return(&auth.Identity{
				UserID: userSubject,
			}, nil).AnyTimes()

			service := NewInviteService()
			invite, err := service.RemoveInvite(ctx, scenario.dBSetup(ctrl), idClient, projectId, userRole, userEmail)

			if scenario.expectedError != "" {
				require.ErrorContains(t, err, scenario.expectedError)
				return
			}
			require.NoError(t, err)

			if scenario.expectedResult != nil {
				require.Equal(t, scenario.expectedResult.Role, invite.Role)
				require.Equal(t, scenario.expectedResult.Project, invite.Project)
			}
		})
	}
}

var (
	projectId   = uuid.New()
	userEmail   = "test@example.com"
	userSubject = "subject"
	userRole    = authz.RoleAdmin
	inviteCode  = "code"
	baseUrl     = "https://minder.example.com"

	validUser = db.User{
		ID:              1,
		IdentitySubject: userSubject,
	}
	noInvites                     []db.GetInvitationsByEmailAndProjectRow
	singleInviteWithDifferentRole = []db.GetInvitationsByEmailAndProjectRow{
		{
			Code:    inviteCode,
			Email:   userEmail,
			Project: projectId,
			Role:    authz.RoleEditor.String(),
		},
	}
	singleInviteWithSameRole = []db.GetInvitationsByEmailAndProjectRow{
		{
			Code:      inviteCode,
			Email:     userEmail,
			Project:   projectId,
			Role:      authz.RoleAdmin.String(),
			UpdatedAt: time.Now().Add(-time.Hour),
		},
	}
	multipleInvites = []db.GetInvitationsByEmailAndProjectRow{
		{
			Email:   userEmail,
			Project: projectId,
			Role:    authz.RoleEditor.String(),
		},
		{
			Email:   userEmail,
			Project: projectId,
			Role:    authz.RoleViewer.String(),
		},
	}

	userInvite = db.UserInvite{
		Code:      inviteCode,
		Project:   projectId,
		Role:      userRole.String(),
		UpdatedAt: time.Now().Add(-time.Minute),
	}
)

func withGetUserBySubject(result db.User) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			GetUserBySubject(gomock.Any(), gomock.Any()).
			Return(result, nil)
	}
}

func withGetUserByID(result db.User) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			GetUserByID(gomock.Any(), gomock.Any()).
			Return(result, nil)
	}
}

func withExistingInvites(result []db.GetInvitationsByEmailAndProjectRow) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			GetInvitationsByEmailAndProject(gomock.Any(), gomock.Any()).
			Return(result, nil)
	}
}

func withInviteRoleUpdate(result db.UserInvite, err error) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			UpdateInvitationRole(gomock.Any(), db.UpdateInvitationRoleParams{
				Code: inviteCode,
				Role: userRole.String(),
			}).
			Return(result, err)
	}
}

func withCreateInvite(result db.UserInvite, err error) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			CreateInvitation(gomock.Any(), gomock.Any()).
			Return(result, err)
	}
}

func withDeleteInvite(result db.UserInvite, err error) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			DeleteInvitation(gomock.Any(), inviteCode).
			Return(result, err)
	}
}

func withProject() func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		projectMetadata, err := json.Marshal(
			projects.Metadata{Public: projects.PublicMetadataV1{}},
		)
		project := db.Project{
			ID:       projectId,
			Metadata: projectMetadata,
		}
		mock.EXPECT().
			GetProjectByID(gomock.Any(), projectId).
			Return(project, err)
	}
}
