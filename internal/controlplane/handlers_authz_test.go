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
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/authz/mock"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/engine"
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
	subject := "subject1"

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
					GetUserBySubject(gomock.Any(), subject).
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

	for i := range testCases {
		tc := testCases[i]
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
			ctx := auth.WithUserSubjectContext(withRpcOptions(context.Background(), rpcOptions), subject)

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
				fmt.Sprintf("user is not authorized to perform this operation on project %q", projectID)),
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

	for i := range testCases {
		tc := testCases[i]
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

func TestListProjects(t *testing.T) {
	t.Parallel()

	user := "testuser"

	authzClient := &mock.SimpleClient{
		Allowed: []uuid.UUID{uuid.New()},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().GetUserBySubject(gomock.Any(), user).Return(db.User{ID: 1}, nil)
	mockStore.EXPECT().GetProjectByID(gomock.Any(), authzClient.Allowed[0]).Return(
		db.Project{ID: authzClient.Allowed[0]}, nil)

	server := Server{
		store:       mockStore,
		authzClient: authzClient,
	}

	ctx := context.Background()
	ctx = auth.WithUserSubjectContext(ctx, user)

	resp, err := server.ListProjects(ctx, &minder.ListProjectsRequest{})
	assert.NoError(t, err)

	assert.Len(t, resp.Projects, 1)
	assert.Equal(t, authzClient.Allowed[0].String(), resp.Projects[0].ProjectId)
}
