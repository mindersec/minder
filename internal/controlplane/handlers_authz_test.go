// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"cmp"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"fmt"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/auth"
	"github.com/mindersec/minder/internal/auth/githubactions"
	authjwt "github.com/mindersec/minder/internal/auth/jwt"
	"github.com/mindersec/minder/internal/auth/jwt/noop"
	"github.com/mindersec/minder/internal/auth/keycloak"
	mockauth "github.com/mindersec/minder/internal/auth/mock"
	"github.com/mindersec/minder/internal/authz"
	"github.com/mindersec/minder/internal/authz/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/engine/engcontext"
	fake "github.com/mindersec/minder/internal/invites/test"
	"github.com/mindersec/minder/internal/roles"
	mockroles "github.com/mindersec/minder/internal/roles/mock"
	"github.com/mindersec/minder/internal/util"
	minder "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
)

// Mock for HasProtoContext
type request struct {
	Context *minder.Context
}

func (m request) GetContext() *minder.Context {
	return m.Context
}

// Reply type containing the detected entityContext.
type replyType struct {
	Context engcontext.EntityContext
}

func TestEntityContextProjectInterceptor(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	otherProjectID := uuid.New()
	defaultProjectID := uuid.New()
	projectIdStr := projectID.String()
	malformedProjectID := "malformed"
	projectName := "my-project"
	//nolint:goconst
	provider := "github"
	userJWT := openid.New()
	assert.NoError(t, userJWT.Set("sub", "subject1"))

	assert.NotEqual(t, projectID, defaultProjectID)

	testCases := []struct {
		name            string
		req             any
		resource        minder.TargetResource
		buildStubs      func(t *testing.T, store *mockdb.MockStore)
		rpcErr          error
		defaultProject  bool
		expectedContext engcontext.EntityContext // Only if non-error
	}{
		{
			name: "not implementing proto context throws error",
			// Does not implement HasProtoContext
			req:      struct{}{},
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			rpcErr:   status.Errorf(codes.Internal, "Error extracting context from request"),
		},
		{
			name:     "target resource unspecified throws error",
			req:      &minder.CheckHealthRequest{},
			resource: minder.TargetResource_TARGET_RESOURCE_UNSPECIFIED,
			rpcErr:   status.Errorf(codes.Internal, "cannot perform authorization, because target resource is unspecified"),
		},
		{
			name:            "non project owner bypasses interceptor",
			req:             &minder.CreateUserRequest{},
			resource:        minder.TargetResource_TARGET_RESOURCE_USER,
			expectedContext: engcontext.EntityContext{},
		},
		{
			name:     "invalid request with nil context and multiple projects",
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			req: &minder.RegisterRepositoryRequest{
				Context: nil,
			},
			buildStubs: func(t *testing.T, store *mockdb.MockStore) {
				t.Helper()
				store.EXPECT().GetUserBySubject(gomock.Any(), userJWT.Subject()).
					Return(db.User{}, nil)
			},
			rpcErr: util.UserVisibleError(codes.InvalidArgument, "Multiple projects found, cannot determine default project."+
				" Please explicitly set a project and run the command again."),
		},
		{
			name: "malformed project ID",
			req: &minder.RegisterRepositoryRequest{
				Context: &minder.Context{
					Project: &malformedProjectID,
				},
			},
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			buildStubs: func(t *testing.T, store *mockdb.MockStore) {
				t.Helper()
				store.EXPECT().
					GetProjectByName(gomock.Any(), malformedProjectID).
					Return(db.Project{}, sql.ErrNoRows)
			},
			rpcErr: util.UserVisibleError(codes.InvalidArgument, `project "malformed" not found`),
		},
		{
			name: "empty context",
			req: &minder.RegisterRepositoryRequest{
				Context: &minder.Context{},
			},
			resource:       minder.TargetResource_TARGET_RESOURCE_PROJECT,
			defaultProject: true,
			buildStubs: func(t *testing.T, store *mockdb.MockStore) {
				t.Helper()
				store.EXPECT().
					GetUserBySubject(gomock.Any(), userJWT.Subject()).
					Return(db.User{
						ID: 1,
					}, nil)
			},
			expectedContext: engcontext.EntityContext{
				// Uses the default project id
				Project: engcontext.Project{ID: defaultProjectID},
			},
		}, {
			name: "no provider",
			req: &minder.CreateProviderRequest{
				Context: &minder.Context{
					Project: &projectIdStr,
				},
			},
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			expectedContext: engcontext.EntityContext{
				Project: engcontext.Project{ID: projectID},
			},
		}, {
			name: "sets entity context",
			req: &minder.RegisterRepositoryRequest{
				Context: &minder.Context{
					Project:  &projectIdStr,
					Provider: &provider,
				},
			},
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			expectedContext: engcontext.EntityContext{
				Project:  engcontext.Project{ID: projectID},
				Provider: engcontext.Provider{Name: provider},
			},
		}, {
			name: "lookup by name",
			req: &minder.AssignRoleRequest{
				Context: &minder.Context{
					Project: &projectName,
				},
			},
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			buildStubs: func(t *testing.T, store *mockdb.MockStore) {
				t.Helper()
				store.EXPECT().
					GetProjectByName(gomock.Any(), projectName).
					Return(db.Project{
						ID:   projectID,
						Name: projectName,
					}, nil)
			},
			expectedContext: engcontext.EntityContext{
				Project: engcontext.Project{ID: projectID},
			},
		}, {
			name: "with ContextV2",
			req: &minder.GetDataSourceByNameRequest{
				Context: &minder.ContextV2{
					ProjectId: projectIdStr,
					Provider:  provider,
				},
			},
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			expectedContext: engcontext.EntityContext{
				Project:  engcontext.Project{ID: projectID},
				Provider: engcontext.Provider{Name: provider},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rpcOptions := &minder.RpcOptions{
				TargetResource: tc.resource,
			}

			unaryHandler := func(ctx context.Context, _ interface{}) (any, error) {
				return replyType{engcontext.EntityFromContext(ctx)}, nil
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			if tc.buildStubs != nil {
				tc.buildStubs(t, mockStore)
			}
			ctx := authjwt.WithAuthTokenContext(withRpcOptions(context.Background(), rpcOptions), userJWT)
			ctx = auth.WithIdentityContext(ctx, &auth.Identity{
				UserID: userJWT.Subject(),
			})

			authzClient := &mock.SimpleClient{}

			if tc.defaultProject {
				authzClient.Allowed = []uuid.UUID{defaultProjectID}
			} else {
				authzClient.Allowed = []uuid.UUID{projectID, otherProjectID}
			}

			server := Server{
				store:       mockStore,
				authzClient: authzClient,
			}
			reply, err := EntityContextProjectInterceptor(ctx, tc.req, &grpc.UnaryServerInfo{
				Server: &server,
			}, unaryHandler)
			if tc.rpcErr != nil {
				assert.Equal(t, tc.rpcErr, err)
				return
			}

			require.NoError(t, err, "expected no error")
			assert.Equal(t, tc.expectedContext, reply.(replyType).Context)
		})
	}
}

func TestProjectAuthorizationInterceptor(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	defaultProjectID := uuid.New()
	userJWT := openid.New()
	assert.NoError(t, userJWT.Set("sub", "subject1"))
	adminJWT := openid.New()
	assert.NoError(t, adminJWT.Set("sub", "admin"))

	assert.NotEqual(t, projectID, defaultProjectID)

	testCases := []struct {
		name      string
		jwt       *openid.Token
		relation  minder.Relation
		entityCtx *engcontext.EntityContext
		resource  minder.TargetResource
		rpcErr    error
	}{
		{
			name:      "anonymous bypasses interceptor",
			entityCtx: &engcontext.EntityContext{},
			resource:  minder.TargetResource_TARGET_RESOURCE_NONE,
		},
		{
			name:      "non project owner bypasses interceptor",
			resource:  minder.TargetResource_TARGET_RESOURCE_USER,
			entityCtx: &engcontext.EntityContext{},
		},
		{
			name:     "not authorized on project error",
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			entityCtx: &engcontext.EntityContext{
				Project: engcontext.Project{
					ID: projectID,
				},
			},
			rpcErr: util.UserVisibleError(
				codes.PermissionDenied,
				"user %q is not authorized to perform this operation on project %q", "subject1", projectID),
		},
		{
			name:     "authorized on project",
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			entityCtx: &engcontext.EntityContext{
				Project: engcontext.Project{
					ID: defaultProjectID,
				},
			},
		},
		{
			name:     "admin create",
			jwt:      &adminJWT,
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			entityCtx: &engcontext.EntityContext{
				Project: engcontext.Project{
					ID: projectID,
				},
			},
			rpcErr: util.UserVisibleError(
				codes.PermissionDenied,
				"user %q is not authorized to perform this operation on project %q", "admin", projectID),
		},
		{
			name:     "admin delete",
			jwt:      &adminJWT,
			relation: minder.Relation_RELATION_PROFILE_DELETE,
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			entityCtx: &engcontext.EntityContext{
				Project: engcontext.Project{
					ID: projectID,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			relation := minder.Relation_RELATION_PROFILE_CREATE
			if tc.relation != minder.Relation_RELATION_UNSPECIFIED {
				relation = tc.relation
			}

			rpcOptions := &minder.RpcOptions{
				TargetResource: tc.resource,
				Relation:       relation,
			}

			unaryHandler := func(ctx context.Context, _ interface{}) (any, error) {
				return replyType{engcontext.EntityFromContext(ctx)}, nil
			}
			server := Server{
				authzClient: &mock.SimpleClient{
					Allowed: []uuid.UUID{defaultProjectID},
				},
				cfg: &serverconfig.Config{
					Authz: serverconfig.AuthzConfig{
						AdminDeleters: []string{"admin"},
					},
				},
			}
			jwt := userJWT
			if tc.jwt != nil {
				jwt = *tc.jwt
			}
			ctx := withRpcOptions(context.Background(), rpcOptions)
			ctx = engcontext.WithEntityContext(ctx, tc.entityCtx)
			ctx = authjwt.WithAuthTokenContext(ctx, jwt)
			ctx = auth.WithIdentityContext(ctx, &auth.Identity{
				UserID:    jwt.Subject(),
				HumanName: jwt.Subject(),
			})
			_, err := ProjectAuthorizationInterceptor(ctx, request{}, &grpc.UnaryServerInfo{
				Server: &server,
			}, unaryHandler)
			if tc.rpcErr != nil {
				assert.Equal(t, tc.rpcErr, err)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRoleManagement(t *testing.T) {
	t.Parallel()

	project := uuid.New()

	user1 := uuid.New()
	user2 := uuid.New()

	tests := []struct {
		name            string
		adds            []*minder.RoleAssignment
		removes         []*minder.RoleAssignment
		expectAddErrors bool
		invites         []db.ListInvitationsForProjectRow
		result          *minder.ListRoleAssignmentsResponse
		stored          []*minder.RoleAssignment
	}{{
		name: "email invitation adds",
		adds: []*minder.RoleAssignment{{
			Role:  authz.RoleAdmin.String(),
			Email: "user1@example.com",
		}, {
			Role:  authz.RoleAdmin.String(),
			Email: "user2@example.com",
		}},
		result: &minder.ListRoleAssignmentsResponse{
			RoleAssignments: []*minder.RoleAssignment{},
		},
	}, {
		name: "email invitation add and remove",
		adds: []*minder.RoleAssignment{{
			Role:  authz.RoleAdmin.String(),
			Email: "user1@example.com",
		}, {
			Role:  authz.RoleAdmin.String(),
			Email: "user2@example.com",
		}},
		removes: []*minder.RoleAssignment{{
			Role:  authz.RoleAdmin.String(),
			Email: "user2@example.com",
		}},
		result: &minder.ListRoleAssignmentsResponse{
			RoleAssignments: []*minder.RoleAssignment{},
		},
	}, {
		name:            "human subject adds rejected",
		expectAddErrors: true,
		adds: []*minder.RoleAssignment{{
			Role:    authz.RoleAdmin.String(),
			Subject: user1.String(),
		}, {
			Role:    authz.RoleAdmin.String(),
			Subject: user2.String(),
		}},
		invites: []db.ListInvitationsForProjectRow{{
			Email:           "george@happyplace.dev",
			Role:            authz.RoleEditor.String(),
			IdentitySubject: user1.String(),
			CreatedAt:       time.Time{},
			UpdatedAt:       time.Time{},
		}},
		result: &minder.ListRoleAssignmentsResponse{
			RoleAssignments: []*minder.RoleAssignment{},
			Invitations: []*minder.Invitation{{
				Role:      authz.RoleEditor.String(),
				Email:     "george@happyplace.dev",
				Project:   project.String(),
				Sponsor:   user1.String(),
				CreatedAt: timestamppb.New(time.Time{}),
				ExpiresAt: timestamppb.New(time.Time{}.Add(7 * 24 * time.Hour)),
				Expired:   true,
			}},
		},
		stored: []*minder.RoleAssignment{},
	}}

	user := openid.New()
	assert.NoError(t, user.Set("sub", "testuser"))

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			authzClient := &mock.SimpleClient{
				Allowed: []uuid.UUID{},
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)

			// Use real implementation, not mock
			roleService := roles.NewRoleService()
			fakeInviteService := fake.NewFakeInviteService()

			mockStore.EXPECT().BeginTransaction().AnyTimes()
			mockStore.EXPECT().GetQuerierWithTransaction(gomock.Any()).Return(mockStore).AnyTimes()
			mockStore.EXPECT().Rollback(gomock.Any()).AnyTimes()
			mockStore.EXPECT().Commit(gomock.Any()).AnyTimes()

			for range tc.adds {
				mockStore.EXPECT().GetProjectByID(gomock.Any(), project).Return(db.Project{ID: project}, nil)
			}

			mockStore.EXPECT().ListInvitationsForProject(gomock.Any(), project).Return(tc.invites, nil)

			server := Server{
				store:       mockStore,
				authzClient: authzClient,
				invites:     fakeInviteService,
				cfg:         &serverconfig.Config{Email: serverconfig.EmailConfig{}},
				idClient: &SimpleResolver{
					data: []auth.Identity{{
						UserID:    user1.String(),
						HumanName: "user1",
						Provider:  &keycloak.KeyCloak{},
					}, {
						UserID:    user2.String(),
						HumanName: "user2",
						Provider:  &keycloak.KeyCloak{},
					}},
				},
				jwt:   noop.NewJwtValidator("test"),
				roles: roleService,
			}

			ctx := context.Background()
			ctx = authjwt.WithAuthTokenContext(ctx, user)
			ctx = auth.WithIdentityContext(ctx, &auth.Identity{
				UserID: "testuser",
			})
			ctx = engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
				Project: engcontext.Project{
					ID: project,
				},
			})
			// Create a signed JWT token with subject and email fields
			privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
			assert.NoError(t, err)
			tokenString, err := createSignedJWTToken("testuser", "testuser@example.com", privateKey)
			assert.NoError(t, err)
			// Set the auth token in the incoming metadata
			md := metadata.Pairs("authorization", "bearer "+tokenString)
			ctx = metadata.NewIncomingContext(ctx, md)

			for _, add := range tc.adds {
				_, err := server.AssignRole(ctx, &minder.AssignRoleRequest{RoleAssignment: add})
				if tc.expectAddErrors {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			}
			for _, remove := range tc.removes {
				_, err := server.RemoveRole(ctx, &minder.RemoveRoleRequest{RoleAssignment: remove})
				assert.NoError(t, err)
			}

			result, err := server.ListRoleAssignments(ctx, &minder.ListRoleAssignmentsRequest{})
			assert.NoError(t, err)

			wantJSON := RoleAssignmentsToJson(t, tc.result.RoleAssignments)
			gotJSON := RoleAssignmentsToJson(t, result.RoleAssignments)
			assert.ElementsMatchf(t, wantJSON, gotJSON, "RPC results mismatch, want: A, got: B")

			wantJSON = InvitationsToJson(t, tc.result.Invitations)
			gotJSON = InvitationsToJson(t, result.Invitations)
			assert.ElementsMatchf(t, wantJSON, gotJSON, "Invitations mismatch, want: A, got: B")

			if len(tc.stored) > 0 {
				wantStored := RoleAssignmentsToJson(t, tc.stored)
				gotStored := RoleAssignmentsToJson(t, authzClient.Assignments[project])
				assert.ElementsMatchf(t, wantStored, gotStored, "Stored results mismatch, want: A, got: B")
			}
		})
	}
}

func TestUpdateRole(t *testing.T) {
	t.Parallel()

	userEmail := "test@example.com"
	authzRole := authz.RoleAdmin
	projectID := uuid.New()

	tests := []struct {
		name               string
		inviteeEmail       string
		subject            string
		expectedError      string
		expectedInvitation bool
		expectedRole       bool
	}{
		{
			name:          "error when self update",
			inviteeEmail:  userEmail,
			expectedError: "cannot update your own role",
		},
		{
			name:          "error with no subject or email",
			expectedError: "one of subject or email must be specified",
		},
		{
			name:               "request with email updates invite",
			inviteeEmail:       "other@example.com",
			expectedInvitation: true,
		},
		{
			name:         "request with subject updates role assignment",
			subject:      "user",
			expectedRole: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			user := openid.New()
			assert.NoError(t, user.Set("email", userEmail))

			ctx := context.Background()
			ctx = authjwt.WithAuthTokenContext(ctx, user)
			// Add Identity context for invite service
			ctx = auth.WithIdentityContext(ctx, &auth.Identity{
				UserID: "testuser",
			})
			// Add entity context with project for UpdateRole endpoint
			ctx = engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
				Project: engcontext.Project{ID: projectID},
			})

			fakeInviteService := fake.NewFakeInviteService()
			// Pre-populate invite for update test
			if tc.expectedInvitation && tc.inviteeEmail != "" {
				_, _ = fakeInviteService.CreateInvite(ctx, nil, nil, serverconfig.EmailConfig{},
					projectID, authzRole, tc.inviteeEmail)
			}
			mockRoleService := mockroles.NewMockRoleService(ctrl)
			if tc.expectedRole {
				mockRoleService.EXPECT().UpdateRoleAssignment(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), tc.subject, authzRole).Return(&minder.RoleAssignment{}, nil)
			}
			mockStore := mockdb.NewMockStore(ctrl)
			mockStore.EXPECT().BeginTransaction().AnyTimes()
			mockStore.EXPECT().GetQuerierWithTransaction(gomock.Any()).AnyTimes()
			mockStore.EXPECT().Commit(gomock.Any()).AnyTimes()
			mockStore.EXPECT().Rollback(gomock.Any()).AnyTimes()

			server := &Server{
				invites: fakeInviteService,
				roles:   mockRoleService,
				store:   mockStore,
				cfg:     &serverconfig.Config{Email: serverconfig.EmailConfig{}},
			}

			response, err := server.UpdateRole(ctx, &minder.UpdateRoleRequest{
				Email:   tc.inviteeEmail,
				Subject: tc.subject,
				Roles:   []string{authzRole.String()},
			})

			if tc.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
				return
			}

			require.NoError(t, err)
			if tc.expectedInvitation {
				require.Equal(t, 1, len(response.Invitations))
			}
			if tc.expectedRole {
				require.Equal(t, 1, len(response.RoleAssignments))
			}
		})
	}
}
func TestAssignRole(t *testing.T) {
	t.Parallel()

	userEmail := "user@test.com"
	authzRole := authz.RoleAdmin
	projectId := uuid.New()
	otherProject := uuid.New()

	tests := []struct {
		name          string
		project       uuid.UUID
		inviteeEmail  string
		subject       string
		buildStubs    func(t *testing.T, store *mockdb.MockStore)
		expectedError string
		userIdentity  *auth.Identity
	}{
		{
			name:          "error with no subject or email",
			expectedError: "one of subject or email must be specified",
		},
		{
			name:          "error when self enroll",
			inviteeEmail:  userEmail,
			expectedError: "cannot update your own role",
		},
		{
			name:          "error when project ID is not found",
			project:       otherProject,
			subject:       "user",
			expectedError: fmt.Sprintf("target project with ID %s not found", otherProject.String()),
		},
		{
			name:         "request with email creates invite",
			inviteeEmail: "other@example.com",
		},
		{
			name:    "request with human subject returns error",
			subject: "user",
			userIdentity: &auth.Identity{
				UserID:   "user",
				Provider: &keycloak.KeyCloak{},
			},
			expectedError: "human users may only be added by invitation",
		}, {
			name:    "grant permission to GitHub Action",
			subject: "githubactions/repo:mindersec/community:ref:refs/heads/main",
			userIdentity: &auth.Identity{
				UserID:   "repo:mindersec/community:ref:refs/heads/main",
				Provider: &githubactions.GitHubActions{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			projectIdString := cmp.Or(tc.project, projectId).String()
			user := openid.New()
			assert.NoError(t, user.Set("email", userEmail))

			ctx := context.Background()
			ctx = authjwt.WithAuthTokenContext(ctx, user)
			// Add Identity context for invite service
			ctx = auth.WithIdentityContext(ctx, &auth.Identity{
				UserID: "testuser",
			})
			ctx = engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
				Project: engcontext.Project{ID: cmp.Or(tc.project, projectId)},
			})

			idClient := mockauth.NewMockResolver(ctrl)
			idClient.EXPECT().Resolve(gomock.Any(), tc.subject).Return(tc.userIdentity, nil).MaxTimes(1)

			fakeInviteService := fake.NewFakeInviteService()
			mockRoleService := mockroles.NewMockRoleService(ctrl)
			if tc.expectedError == "" && tc.userIdentity != nil {
				mockRoleService.EXPECT().CreateRoleAssignment(gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), *tc.userIdentity, authzRole).Return(&minder.RoleAssignment{
					Role:    authzRole.String(),
					Project: &projectIdString,
				}, nil)
			}

			mockStore := mockdb.NewMockStore(ctrl)
			// Most tests will call GetProjectByID with the correct ID, but some will
			// deliberately choose a different ID; return an error for those.
			mockStore.EXPECT().GetProjectByID(gomock.Any(), projectId).Return(db.Project{
				ID: projectId,
			}, nil).MaxTimes(1)
			mockStore.EXPECT().GetProjectByID(gomock.Any(), gomock.Not(gomock.Eq(projectId))).
				Return(db.Project{}, sql.ErrNoRows).MaxTimes(1)
			mockStore.EXPECT().BeginTransaction().AnyTimes()
			mockStore.EXPECT().GetQuerierWithTransaction(gomock.Any()).AnyTimes()
			mockStore.EXPECT().Commit(gomock.Any()).AnyTimes()
			mockStore.EXPECT().Rollback(gomock.Any()).AnyTimes()
			if tc.buildStubs != nil {
				tc.buildStubs(t, mockStore)
			}

			server := &Server{
				invites:  fakeInviteService,
				roles:    mockRoleService,
				store:    mockStore,
				idClient: idClient,
				cfg:      &serverconfig.Config{Email: serverconfig.EmailConfig{}},
			}

			response, err := server.AssignRole(ctx, &minder.AssignRoleRequest{
				Context: &minder.Context{
					Project: &projectIdString,
				},
				RoleAssignment: &minder.RoleAssignment{
					Role:    authzRole.String(),
					Subject: tc.subject,
					Email:   tc.inviteeEmail,
				},
			})

			if tc.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
				return
			}

			require.NoError(t, err)
			if tc.userIdentity != nil {
				require.Equal(t, authzRole.String(), response.RoleAssignment.Role)
			} else {
				require.Equal(t, authzRole.String(), response.Invitation.Role)
				require.Equal(t, projectIdString, response.Invitation.Project)
			}
		})
	}
}

func TestRemoveRole(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	projectIdString := projectID.String()
	userEmail := "test@example.com"
	authzRole := authz.RoleAdmin

	tests := []struct {
		name               string
		inviteeEmail       string
		subject            string
		expectedError      string
		expectedInvitation bool
		expectedRole       bool
	}{
		{
			name:          "error with no subject or email",
			expectedError: "one of subject or email must be specified",
		},
		{
			name:               "request with email deletes invite",
			inviteeEmail:       "other@example.com",
			expectedInvitation: true,
		},
		{
			name:          "no invite present produces error",
			inviteeEmail:  "no-invite@example.com",
			expectedError: "no invitation found for email no-invite@example.com",
		},
		{
			name:         "request with subject deletes role assignment",
			subject:      "user",
			expectedRole: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			user := openid.New()
			assert.NoError(t, user.Set("email", userEmail))

			ctx := context.Background()
			ctx = authjwt.WithAuthTokenContext(ctx, user)
			// Add Identity context for invite service
			ctx = auth.WithIdentityContext(ctx, &auth.Identity{
				UserID: "testuser",
			})
			// Add entity context with project for RemoveRole endpoint
			ctx = engcontext.WithEntityContext(ctx, &engcontext.EntityContext{
				Project: engcontext.Project{ID: projectID},
			})

			fakeInviteService := fake.NewFakeInviteService()
			// Pre-populate invite for delete test
			if tc.expectedInvitation && tc.inviteeEmail != "" {
				_, _ = fakeInviteService.CreateInvite(ctx, nil, nil, serverconfig.EmailConfig{},
					projectID, authzRole, tc.inviteeEmail)
			}
			mockRoleService := mockroles.NewMockRoleService(ctrl)
			if tc.expectedRole {
				mockRoleService.EXPECT().RemoveRoleAssignment(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
					gomock.Any(), tc.subject, authzRole).Return(&minder.RoleAssignment{
					Role:    authzRole.String(),
					Project: &projectIdString,
				}, nil)
			}
			mockStore := mockdb.NewMockStore(ctrl)
			mockStore.EXPECT().BeginTransaction().AnyTimes()
			mockStore.EXPECT().GetQuerierWithTransaction(gomock.Any()).AnyTimes()
			mockStore.EXPECT().Commit(gomock.Any()).AnyTimes()
			mockStore.EXPECT().Rollback(gomock.Any()).AnyTimes()

			server := &Server{
				invites: fakeInviteService,
				roles:   mockRoleService,
				store:   mockStore,
				cfg:     &serverconfig.Config{Email: serverconfig.EmailConfig{}},
			}

			response, err := server.RemoveRole(ctx, &minder.RemoveRoleRequest{
				Context: &minder.Context{
					Project: &projectIdString,
				},
				RoleAssignment: &minder.RoleAssignment{
					Role:    authzRole.String(),
					Subject: tc.subject,
					Email:   tc.inviteeEmail,
				},
			})

			if tc.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedError)
				return
			}

			require.NoError(t, err)
			if tc.expectedInvitation {
				require.Equal(t, authzRole.String(), response.Invitation.Role)
				require.Equal(t, projectIdString, response.Invitation.Project)
			}
			if tc.expectedRole {
				require.Equal(t, authzRole.String(), response.RoleAssignment.Role)
			}
		})
	}
}

func RoleAssignmentsToJson(t *testing.T, assignments []*minder.RoleAssignment) []string {
	t.Helper()
	json := make([]string, 0, len(assignments))
	for _, p := range assignments {
		j, err := protojson.Marshal(p)
		assert.NoError(t, err)
		json = append(json, string(j))
	}
	return json
}

func InvitationsToJson(t *testing.T, invites []*minder.Invitation) []string {
	t.Helper()
	json := make([]string, 0, len(invites))
	for _, p := range invites {
		j, err := protojson.Marshal(p)
		assert.NoError(t, err)
		json = append(json, string(j))
	}
	return json
}

type SimpleResolver struct {
	data []auth.Identity
}

var _ auth.Resolver = (*SimpleResolver)(nil)

// Resolve implements auth.Resolver.
func (s *SimpleResolver) Resolve(_ context.Context, id string) (*auth.Identity, error) {
	for _, i := range s.data {
		if i.UserID == id {
			return &i, nil
		}
		if i.HumanName == id {
			return &i, nil
		}
	}
	return nil, fmt.Errorf("user %q not found", id)
}

// Validate implements auth.Resolver.
func (*SimpleResolver) Validate(_ context.Context, _ jwt.Token) (*auth.Identity, error) {
	panic("unimplemented")
}

// createSignedJWTToken creates a signed JWT token with the specified subject and email.
func createSignedJWTToken(subject, email string, privateKey *rsa.PrivateKey) (string, error) {
	token := gojwt.NewWithClaims(gojwt.SigningMethodRS256, gojwt.MapClaims{
		"sub":   subject,
		"email": email,
		"exp":   time.Now().Add(time.Hour * 1).Unix(),
	})
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
