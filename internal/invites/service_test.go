// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package invites

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/auth"
	"github.com/mindersec/minder/internal/auth/githubactions"
	authjwt "github.com/mindersec/minder/internal/auth/jwt"
	mockauth "github.com/mindersec/minder/internal/auth/mock"
	"github.com/mindersec/minder/internal/authz"
	"github.com/mindersec/minder/internal/db"
	dbf "github.com/mindersec/minder/internal/db/fixtures"
	"github.com/mindersec/minder/internal/email"
	"github.com/mindersec/minder/internal/projects"
	minder "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/config/server"
	mockevents "github.com/mindersec/minder/pkg/eventer/interfaces/mock"
)

func TestMain(m *testing.M) {
	// Needed for loading email templates
	err := os.Setenv("KO_DATA_PATH", "../../cmd/server/kodata")
	if err != nil {
		fmt.Printf("error setting KO_DATA_PATH: %v\n", err)
		os.Exit(1)
	}
	projectId := uuid.New()
	_, err = email.NewMessage(context.Background(), "j@example.com", "ABC123", "http://example.com/invite", "http://api.example.com/", "admin", projectId, "Example", "Joe")
	if err != nil {
		fmt.Printf("error creating message: %v\n", err)
		os.Exit(1)
	}

	m.Run()
}

func TestCreateInvite(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name           string
		idProvider     auth.IdentityProvider
		dBSetup        dbf.DBMockBuilder
		publisherSetup func(t *testing.T, pub *mockevents.MockPublisher)
		expectedError  string
		expectedResult *minder.Invitation
	}{
		{
			name: "error when existing invites",
			dBSetup: dbf.NewDBMock(
				withGetUserBySubject(validUser),
				withExistingInvites(multipleInvites...),
			),
			expectedError: "invitation for this email and project already exists, use update instead",
		},
		{
			name: "invite created and message sent successfully",
			dBSetup: dbf.NewDBMock(
				withGetUserBySubject(validUser),
				withExistingInvites(),
				withCreateInvite(userInvite, nil),
				withProject(nil),
			),
			publisherSetup: func(_ *testing.T, pub *mockevents.MockPublisher) {
				pub.EXPECT().Publish(email.TopicQueueInviteEmail, gomock.Any())
			},
			expectedResult: &minder.Invitation{
				Project: projectId.String(),
				Role:    userRole.String(),
			},
		},
		{
			name:          "service account can't create invite",
			idProvider:    &githubactions.GitHubActions{},
			dBSetup:       dbf.NewDBMock(),
			expectedError: "only human users can create invites",
		},
		{
			name: "attempted injection",
			dBSetup: dbf.NewDBMock(
				withGetUserBySubject(validUser),
				withExistingInvites(),
				withCreateInvite(userInvite, nil),
				withProject(&projects.PublicMetadataV1{
					Description: "<script>alert('xss')</script>",
					DisplayName: "<script>alert('xss')</script>",
				}),
			),
			expectedError: "contains HTML injection",
		},
		{
			name: "bad name",
			dBSetup: dbf.NewDBMock(
				withGetUserBySubject(validUser),
				withExistingInvites(),
				withCreateInvite(userInvite, nil),
				withProject(&projects.PublicMetadataV1{
					Description: "<script>alert('xss')</script>",
					DisplayName: "<script>alert('xss')</script>",
				}),
			),
			expectedError: "Description: Invalid argument\nDetails: error creating email message",
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
			ctx = auth.WithIdentityContext(ctx, &auth.Identity{
				UserID:   userSubject,
				Provider: scenario.idProvider,
			})

			publisher := mockevents.NewMockPublisher(ctrl)
			if scenario.publisherSetup != nil {
				scenario.publisherSetup(t, publisher)
			}

			emailConfig := server.EmailConfig{
				MinderURLBase: baseUrl,
			}

			service := NewInviteService()
			invite, err := service.CreateInvite(ctx, scenario.dBSetup(ctrl), publisher, emailConfig, projectId, userRole, userEmail)

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

	inviteWithin24h := singleInviteWithSameRole
	inviteWithin24h.UpdatedAt = time.Now().Add(-5 * time.Hour)

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
				withExistingInvites(),
			),
			expectedError: "no invitations found for this email and project",
		},
		{
			name: "error when multiple existing invites",
			dBSetup: dbf.NewDBMock(
				withExistingInvites(multipleInvites...),
			),
			expectedError: "multiple invitations found for this email and project",
		},
		{
			name: "no message sent when role is the same",
			dBSetup: dbf.NewDBMock(
				withExistingInvites(singleInviteWithSameRole),
				withInviteRoleUpdate(userInvite),
				withProject(nil),
			),
		},
		{
			name: "invite updated and message sent successfully",
			dBSetup: dbf.NewDBMock(
				withExistingInvites(singleInviteWithDifferentRole),
				withInviteRoleUpdate(userInvite),
				withProject(nil),
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
		{
			name: "no extra emails within 24h",
			dBSetup: dbf.NewDBMock(
				withExistingInvites(inviteWithin24h),
				withInviteRoleUpdate(userInvite),
				withProject(nil),
			),
			expectedResult: &minder.Invitation{
				Project:      projectId.String(),
				Role:         userRole.String(),
				InviteUrl:    fmt.Sprintf("%s/join/%s", baseUrl, inviteCode),
				EmailSkipped: true,
			},
		},
		{
			name: "attempted injection",
			dBSetup: dbf.NewDBMock(
				withExistingInvites(singleInviteWithDifferentRole),
				withInviteRoleUpdate(userInvite),
				withProject(&projects.PublicMetadataV1{
					Description: "<script>alert('xss')</script>",
					DisplayName: "<script>alert('xss')</script>",
				}),
			),
			expectedError: "contains HTML injection",
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
			// ctx = authjwt.WithAuthTokenContext(ctx, user)
			ctx = auth.WithIdentityContext(ctx, &auth.Identity{
				UserID: userSubject,
			})

			publisher := mockevents.NewMockPublisher(ctrl)
			if scenario.publisherSetup != nil {
				scenario.publisherSetup(t, publisher)
			}

			emailConfig := server.EmailConfig{
				MinderURLBase: baseUrl,
			}

			service := NewInviteService()
			invite, err := service.UpdateInvite(ctx, scenario.dBSetup(ctrl), publisher, emailConfig, projectId, userRole, userEmail)

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
		{ // These technically test GetInvitesForEmail as well, following the implementation in handlers_users.go
			name: "error when no existing invites",
			dBSetup: dbf.NewDBMock(
				withExistingInvites(),
			),
			expectedError: "no invitation found",
		},
		{
			name: "error when no invite matches role",
			dBSetup: dbf.NewDBMock(
				withExistingInvites(singleInviteWithDifferentRole),
			),
			expectedError: "no invitation found",
		},
		{
			name: "no message sent when role is the same",
			dBSetup: dbf.NewDBMock(
				withExistingInvites(singleInviteWithSameRole),
				withDeleteInvite(userInvite, nil),
			),
		},
		{
			name: "remove invite failed if no invite fond",
			dBSetup: dbf.NewDBMock(
				// To get through test harness
				withExistingInvites(singleInviteWithSameRole),
				withDeleteInvite(db.UserInvite{}, errors.New("rows not found")),
			),
			expectedError: "error deleting invitation: rows not found",
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
			dbMock := scenario.dBSetup(ctrl)

			var invite *minder.Invitation
			service := NewInviteService()
			invites, err := service.GetInvitesForEmail(ctx, dbMock, projectId, userEmail)
			require.NoError(t, err)
			selected := slices.IndexFunc(invites, func(m *minder.Invitation) bool {
				return m.Role == userRole.String()
			})
			if selected == -1 {
				err = errors.New("no invitation found")
			} else {
				invite = invites[selected]
				err = service.RemoveInvite(ctx, dbMock, invite.Code)
			}

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

func TestGetInvitesForSelf(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name          string
		dBSetup       dbf.DBMockBuilder
		callCtx       func(context.Context) context.Context
		expectedError string
		expectedCount int
	}{
		{
			name: "no invites found",
			dBSetup: dbf.NewDBMock(
				func(mock dbf.DBMock) {
					mock.EXPECT().
						GetInvitationsByEmail(gomock.Any(), userEmail).
						Return([]db.GetInvitationsByEmailRow{}, nil)
				},
			),
			expectedCount: 0,
		},
		{
			name: "multiple invites found",
			dBSetup: dbf.NewDBMock(
				func(mock dbf.DBMock) {
					projectMetadata, _ := json.Marshal(
						projects.Metadata{Public: projects.PublicMetadataV1{DisplayName: "Test Project"}},
					)
					project2ID := uuid.New()
					mock.EXPECT().
						GetInvitationsByEmail(gomock.Any(), userEmail).
						Return([]db.GetInvitationsByEmailRow{
							{Code: "code1", Email: userEmail, Project: projectId, Role: authz.RoleEditor.String(), IdentitySubject: userSubject},
							{Code: "code2", Email: userEmail, Project: project2ID, Role: authz.RoleViewer.String(), IdentitySubject: userSubject},
						}, nil)
					mock.EXPECT().
						GetProjectByID(gomock.Any(), projectId).
						Return(db.Project{ID: projectId, Metadata: projectMetadata}, nil)
					mock.EXPECT().
						GetProjectByID(gomock.Any(), project2ID).
						Return(db.Project{ID: project2ID, Metadata: projectMetadata}, nil)
				},
			),
			expectedCount: 2,
		},
		{
			name:    "no user in context",
			dBSetup: dbf.NewDBMock(),
			callCtx: func(context.Context) context.Context {
				return context.Background()
			},
			expectedError: "failed to get user email",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			user := openid.New()
			assert.NoError(t, user.Set("email", userEmail))

			ctx := context.Background()
			ctx = authjwt.WithAuthTokenContext(ctx, user)
			if scenario.callCtx != nil {
				ctx = scenario.callCtx(ctx)
			}

			idClient := mockauth.NewMockResolver(ctrl)
			idClient.EXPECT().Resolve(ctx, userSubject).Return(&auth.Identity{
				UserID: userSubject,
			}, nil).AnyTimes()

			service := NewInviteService()
			invites, err := service.GetInvitesForSelf(ctx, scenario.dBSetup(ctrl), idClient)

			if scenario.expectedError != "" {
				require.ErrorContains(t, err, scenario.expectedError)
				return
			}
			require.NoError(t, err)
			require.Len(t, invites, scenario.expectedCount)
		})
	}
}

func TestGetInvite(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name           string
		dBSetup        dbf.DBMockBuilder
		idProvider     auth.IdentityProvider
		code           string
		expectedError  string
		expectedResult *minder.Invitation
	}{
		{
			name: "error when invite not found",
			dBSetup: dbf.NewDBMock(
				func(mock dbf.DBMock) {
					mock.EXPECT().
						GetInvitationByCode(gomock.Any(), "invalid-code").
						Return(db.GetInvitationByCodeRow{}, sql.ErrNoRows)
				},
			),
			code:          "invalid-code",
			expectedError: "invitation not found or already used",
		},
		{
			name: "invite retrieved successfully",
			dBSetup: dbf.NewDBMock(
				func(mock dbf.DBMock) {
					mock.EXPECT().GetInvitationByCode(gomock.Any(), inviteCode).
						Return(db.GetInvitationByCodeRow{
							Code:            inviteCode,
							Email:           userEmail,
							Project:         projectId,
							Role:            userRole.String(),
							IdentitySubject: userSubject,
							Sponsor:         2, // Different from current user
							UpdatedAt:       time.Now(),
						}, nil)
				},
				withGetUserBySubject(validUser),
			),
			code: inviteCode,
			expectedResult: &minder.Invitation{
				Code:    inviteCode,
				Email:   userEmail,
				Project: projectId.String(),
				Role:    userRole.String(),
			},
		},
		{
			name:       "service account can't get invite",
			idProvider: &githubactions.GitHubActions{},
			dBSetup: dbf.NewDBMock(
				func(mock dbf.DBMock) {
					mock.EXPECT().GetInvitationByCode(gomock.Any(), inviteCode).
						Return(db.GetInvitationByCodeRow{
							Code:            inviteCode,
							Email:           userEmail,
							Project:         projectId,
							Role:            userRole.String(),
							IdentitySubject: userSubject,
							Sponsor:         2, // Different from current user
							UpdatedAt:       time.Now(),
						}, nil)
				},
			),
			code:          inviteCode,
			expectedError: "this type of user cannot use invitations",
		},
		{
			name: "sponsor is the same as current user",
			dBSetup: dbf.NewDBMock(
				func(mock dbf.DBMock) {
					mock.EXPECT().GetInvitationByCode(gomock.Any(), inviteCode).
						Return(db.GetInvitationByCodeRow{
							Code:            inviteCode,
							Email:           userEmail,
							Project:         projectId,
							Role:            userRole.String(),
							IdentitySubject: userSubject,
							Sponsor:         1, // Same as current user
							UpdatedAt:       time.Now(),
						}, nil)
				},
				withGetUserBySubject(validUser),
			),
			code:          inviteCode,
			expectedError: "users cannot accept their own invitations",
		},
		{
			name: "expired invitation",
			dBSetup: dbf.NewDBMock(
				func(mock dbf.DBMock) {
					mock.EXPECT().GetInvitationByCode(gomock.Any(), inviteCode).
						Return(db.GetInvitationByCodeRow{
							Code:            inviteCode,
							Email:           userEmail,
							Project:         projectId,
							Role:            userRole.String(),
							IdentitySubject: userSubject,
							Sponsor:         2,
							UpdatedAt:       time.Now().Add(-10 * 24 * time.Hour),
						}, nil)
				},
				withGetUserBySubject(validUser),
			),
			code:          inviteCode,
			expectedError: "Description: Permission denied\nDetails: invitation expired",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()
			ctx = auth.WithIdentityContext(ctx, &auth.Identity{
				UserID:   userSubject,
				Provider: scenario.idProvider,
			})

			service := NewInviteService()
			invite, err := service.GetInvite(ctx, scenario.dBSetup(ctrl), scenario.code)

			if scenario.expectedError != "" {
				require.ErrorContains(t, err, scenario.expectedError)
				return
			}
			require.NoError(t, err)

			if scenario.expectedResult != nil {
				require.Equal(t, scenario.expectedResult.Code, invite.Code)
				require.Equal(t, scenario.expectedResult.Email, invite.Email)
				require.Equal(t, scenario.expectedResult.Project, invite.Project)
				require.Equal(t, scenario.expectedResult.Role, invite.Role)
			}
		})
	}
}

func TestGetInvitesForEmail(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name          string
		dBSetup       dbf.DBMockBuilder
		expectedError string
		expectedCount int
	}{
		{
			name: "no invites found",
			dBSetup: dbf.NewDBMock(
				withExistingInvites(),
			),
			expectedCount: 0,
		},
		{
			name: "single invite found",
			dBSetup: dbf.NewDBMock(
				withExistingInvites(singleInviteWithSameRole),
			),
			expectedCount: 1,
		},
		{
			name: "multiple invites found",
			dBSetup: dbf.NewDBMock(
				withExistingInvites(multipleInvites...),
			),
			expectedCount: 2,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()

			service := NewInviteService()
			invites, err := service.GetInvitesForEmail(ctx, scenario.dBSetup(ctrl), projectId, userEmail)

			if scenario.expectedError != "" {
				require.ErrorContains(t, err, scenario.expectedError)
				return
			}
			require.NoError(t, err)
			require.Len(t, invites, scenario.expectedCount)
		})
	}
}

var (
	projectId   = uuid.New()
	userEmail   = "test@example.com"
	userSubject = uuid.New().String()
	userRole    = authz.RoleAdmin
	inviteCode  = "code"
	baseUrl     = "https://minder.example.com"

	validUser = db.User{
		ID:              1,
		IdentitySubject: userSubject,
	}
	singleInviteWithDifferentRole = db.GetInvitationsByEmailAndProjectRow{
		Code:    inviteCode,
		Email:   userEmail,
		Project: projectId,
		Role:    authz.RoleEditor.String(),
	}
	singleInviteWithSameRole = db.GetInvitationsByEmailAndProjectRow{
		Code:      inviteCode,
		Email:     userEmail,
		Project:   projectId,
		Role:      authz.RoleAdmin.String(),
		UpdatedAt: time.Now().Add(-time.Hour),
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

func withExistingInvites(result ...db.GetInvitationsByEmailAndProjectRow) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			GetInvitationsByEmailAndProject(gomock.Any(), gomock.Any()).
			Return(result, nil)
	}
}

func withInviteRoleUpdate(result db.UserInvite) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		mock.EXPECT().
			UpdateInvitationRole(gomock.Any(), db.UpdateInvitationRoleParams{
				Code: inviteCode,
				Role: userRole.String(),
			}).
			Return(result, nil)
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

func withProject(md *projects.PublicMetadataV1) func(dbf.DBMock) {
	return func(mock dbf.DBMock) {
		if md == nil {
			md = &projects.PublicMetadataV1{}
		}
		projectMetadata, err := json.Marshal(
			projects.Metadata{Public: *md},
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
