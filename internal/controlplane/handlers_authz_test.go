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
	"crypto/rand"
	"crypto/rsa"
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
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/auth/noop"
	"github.com/stacklok/minder/internal/authz"
	"github.com/stacklok/minder/internal/authz/mock"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/flags"
	"github.com/stacklok/minder/internal/util"
	minder "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
	Context engine.EntityContext
}

func TestEntityContextProjectInterceptor(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	defaultProjectID := uuid.New()
	projectIdStr := projectID.String()
	malformedProjectID := "malformed"
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
		expectedContext engine.EntityContext // Only if non-error
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
			req:      &request{},
			resource: minder.TargetResource_TARGET_RESOURCE_UNSPECIFIED,
			rpcErr:   status.Errorf(codes.Internal, "cannot perform authorization, because target resource is unspecified"),
		},
		{
			name:            "non project owner bypasses interceptor",
			req:             &request{},
			resource:        minder.TargetResource_TARGET_RESOURCE_USER,
			expectedContext: engine.EntityContext{},
		},
		{
			name:     "invalid request with nil context",
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			req: &request{
				Context: nil,
			},
			rpcErr: util.UserVisibleError(codes.InvalidArgument, "context cannot be nil"),
		},
		{
			name: "malformed project ID",
			req: &request{
				Context: &minder.Context{
					Project: &malformedProjectID,
				},
			},
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			rpcErr:   util.UserVisibleError(codes.InvalidArgument, "malformed project ID"),
		},
		{
			name: "empty context",
			req: &request{
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
			expectedContext: engine.EntityContext{
				// Uses the default project id
				Project: engine.Project{ID: defaultProjectID},
			},
		}, {
			name: "no provider",
			req: &request{
				Context: &minder.Context{
					Project: &projectIdStr,
				},
			},
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			expectedContext: engine.EntityContext{
				Project: engine.Project{ID: projectID},
			},
		}, {
			name: "sets entity context",
			req: &request{
				Context: &minder.Context{
					Project:  &projectIdStr,
					Provider: &provider,
				},
			},
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			expectedContext: engine.EntityContext{
				Project:  engine.Project{ID: projectID},
				Provider: engine.Provider{Name: provider},
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
				return replyType{engine.EntityFromContext(ctx)}, nil
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mockdb.NewMockStore(ctrl)
			if tc.buildStubs != nil {
				tc.buildStubs(t, mockStore)
			}
			ctx := auth.WithAuthTokenContext(withRpcOptions(context.Background(), rpcOptions), userJWT)

			authzClient := &mock.SimpleClient{}

			if tc.defaultProject {
				authzClient.Allowed = []uuid.UUID{defaultProjectID}
			} else {
				authzClient.Allowed = []uuid.UUID{projectID}
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

	assert.NotEqual(t, projectID, defaultProjectID)

	testCases := []struct {
		name      string
		entityCtx *engine.EntityContext
		resource  minder.TargetResource
		rpcErr    error
	}{
		{
			name:      "anonymous bypasses interceptor",
			entityCtx: &engine.EntityContext{},
			resource:  minder.TargetResource_TARGET_RESOURCE_NONE,
		},
		{
			name:      "non project owner bypasses interceptor",
			resource:  minder.TargetResource_TARGET_RESOURCE_USER,
			entityCtx: &engine.EntityContext{},
		},
		{
			name:     "not authorized on project error",
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			entityCtx: &engine.EntityContext{
				Project: engine.Project{
					ID: projectID,
				},
			},
			rpcErr: util.UserVisibleError(
				codes.PermissionDenied,
				fmt.Sprintf("user %q is not authorized to perform this operation on project %q", "subject1", projectID)),
		},
		{
			name:     "authorized on project",
			resource: minder.TargetResource_TARGET_RESOURCE_PROJECT,
			entityCtx: &engine.EntityContext{
				Project: engine.Project{
					ID: defaultProjectID,
				},
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
				return replyType{engine.EntityFromContext(ctx)}, nil
			}
			server := Server{
				authzClient: &mock.SimpleClient{
					Allowed: []uuid.UUID{defaultProjectID},
				},
			}
			ctx := withRpcOptions(context.Background(), rpcOptions)
			ctx = engine.WithEntityContext(ctx, tc.entityCtx)
			ctx = auth.WithAuthTokenContext(ctx, userJWT)
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
		name               string
		idpFlag            bool
		userManagementFlag bool
		adds               []*minder.RoleAssignment
		removes            []*minder.RoleAssignment
		invites            []db.ListInvitationsForProjectRow
		result             *minder.ListRoleAssignmentsResponse
		stored             []*minder.RoleAssignment
	}{{
		name: "simple adds",
		adds: []*minder.RoleAssignment{{
			Role:    authz.AuthzRoleAdmin.String(),
			Subject: user1.String(),
		}, {
			Role:    authz.AuthzRoleAdmin.String(),
			Subject: user2.String(),
		}},
		result: &minder.ListRoleAssignmentsResponse{
			RoleAssignments: []*minder.RoleAssignment{{
				Role:        authz.AuthzRoleAdmin.String(),
				Subject:     user1.String(),
				DisplayName: "user1",
				Project:     proto.String(project.String()),
			}, {
				Role:        authz.AuthzRoleAdmin.String(),
				DisplayName: "user2",
				Subject:     user2.String(),
				Project:     proto.String(project.String()),
			}},
		},
		stored: []*minder.RoleAssignment{{
			Role:    authz.AuthzRoleAdmin.String(),
			Subject: user1.String(),
			Project: proto.String(project.String()),
		}, {
			Role:    authz.AuthzRoleAdmin.String(),
			Subject: user2.String(),
			Project: proto.String(project.String()),
		}},
	}, {
		name: "add and remove",
		adds: []*minder.RoleAssignment{{
			Role:    authz.AuthzRoleAdmin.String(),
			Subject: user1.String(),
		}, {
			Role:    authz.AuthzRoleAdmin.String(),
			Subject: user2.String(),
		}},
		removes: []*minder.RoleAssignment{{
			Role:    authz.AuthzRoleAdmin.String(),
			Subject: user2.String(),
		}},
		result: &minder.ListRoleAssignmentsResponse{
			RoleAssignments: []*minder.RoleAssignment{{
				Role:        authz.AuthzRoleAdmin.String(),
				DisplayName: "user1",
				Subject:     user1.String(),
				Project:     proto.String(project.String()),
			}},
		},
	}, {
		name:    "IDP resolution",
		idpFlag: true,
		adds: []*minder.RoleAssignment{{
			Role:    authz.AuthzRoleAdmin.String(),
			Subject: "user1",
		}, {
			Role:    authz.AuthzRoleAdmin.String(),
			Subject: user2.String(),
		}},
		result: &minder.ListRoleAssignmentsResponse{
			RoleAssignments: []*minder.RoleAssignment{{
				Role:        authz.AuthzRoleAdmin.String(),
				Subject:     user1.String(),
				DisplayName: "user1",
				Project:     proto.String(project.String()),
			}, {
				Role:        authz.AuthzRoleAdmin.String(),
				Subject:     user2.String(),
				DisplayName: "user2",
				Project:     proto.String(project.String()),
			}},
		},
		stored: []*minder.RoleAssignment{{
			Role:    authz.AuthzRoleAdmin.String(),
			Subject: user1.String(),
			Project: proto.String(project.String()),
		}, {
			Role:    authz.AuthzRoleAdmin.String(),
			Subject: user2.String(),
			Project: proto.String(project.String()),
		}},
	}, {
		name: "User Management enabled",
		// NOTE: we don't have a way to create invitations yet.
		userManagementFlag: true,
		adds: []*minder.RoleAssignment{{
			Role:    authz.AuthzRoleAdmin.String(),
			Subject: user1.String(),
		}, {
			Role:    authz.AuthzRoleAdmin.String(),
			Subject: user2.String(),
		}},
		invites: []db.ListInvitationsForProjectRow{{
			Email:           "george@happyplace.dev",
			Role:            authz.AuthzRoleEditor.String(),
			IdentitySubject: user1.String(),
			CreatedAt:       time.Time{},
			UpdatedAt:       time.Time{},
		}},
		result: &minder.ListRoleAssignmentsResponse{
			RoleAssignments: []*minder.RoleAssignment{{
				Role:        authz.AuthzRoleAdmin.String(),
				Subject:     user1.String(),
				DisplayName: "user1",
				Project:     proto.String(project.String()),
			}, {
				Role:        authz.AuthzRoleAdmin.String(),
				Subject:     user2.String(),
				DisplayName: "user2",
				Project:     proto.String(project.String()),
			}},
			Invitations: []*minder.Invitation{{
				Role:           authz.AuthzRoleEditor.String(),
				Email:          "george@happyplace.dev",
				Project:        project.String(),
				Sponsor:        user1.String(),
				SponsorDisplay: "user1",
				CreatedAt:      timestamppb.New(time.Time{}),
				ExpiresAt:      timestamppb.New(time.Time{}.Add(7 * 24 * time.Hour)),
				Expired:        true,
			}},
		},
		stored: []*minder.RoleAssignment{{
			Role:    authz.AuthzRoleAdmin.String(),
			Subject: user1.String(),
			Project: proto.String(project.String()),
		}, {
			Role:    authz.AuthzRoleAdmin.String(),
			Subject: user2.String(),
			Project: proto.String(project.String()),
		}},
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

			for _, add := range tc.adds {
				match := gomock.Eq(add.GetSubject())
				if tc.idpFlag {
					// note: in the flag case, subject may be translated to UUID.
					match = gomock.Any()
				}
				mockStore.EXPECT().GetUserBySubject(gomock.Any(), match).Return(db.User{ID: 1}, nil)
				mockStore.EXPECT().GetProjectByID(gomock.Any(), project).Return(db.Project{ID: project}, nil)

			}
			for _, remove := range tc.removes {
				match := gomock.Eq(remove.GetSubject())
				if tc.idpFlag {
					// note: in the flag case, subject may be translated to UUID.
					match = gomock.Any()
				}
				mockStore.EXPECT().GetUserBySubject(gomock.Any(), match).Return(db.User{ID: 1}, nil)
			}
			if tc.userManagementFlag {
				mockStore.EXPECT().ListInvitationsForProject(gomock.Any(), project).Return(tc.invites, nil)
			}

			featureClient := &flags.FakeClient{}
			featureClient.Data = map[string]any{
				"user_management": tc.userManagementFlag,
			}

			server := Server{
				store:        mockStore,
				authzClient:  authzClient,
				featureFlags: featureClient,
				idClient: &SimpleResolver{
					data: []auth.Identity{{
						UserID:    user1.String(),
						HumanName: "user1",
					}, {
						UserID:    user2.String(),
						HumanName: "user2",
					}},
				},
				jwt: noop.NewJwtValidator("test"),
			}

			ctx := context.Background()
			ctx = auth.WithAuthTokenContext(ctx, user)
			ctx = engine.WithEntityContext(ctx, &engine.EntityContext{
				Project: engine.Project{
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
				assert.NoError(t, err)
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
func (_ *SimpleResolver) Validate(_ context.Context, _ jwt.Token) (*auth.Identity, error) {
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
