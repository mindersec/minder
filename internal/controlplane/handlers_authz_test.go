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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/auth"
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
	//nolint:goconst
	provider := "github"

	assert.NotEqual(t, projectID, defaultProjectID)

	testCases := []struct {
		name            string
		req             any
		scope           minder.ObjectOwner
		rpcErr          error
		expectedContext engine.EntityContext // Only if non-error
	}{
		{
			name: "non project owner bypasses interceptor",
			// Does not implement HasProtoContext
			req:    struct{}{},
			rpcErr: status.Errorf(codes.Internal, "Error extracting context from request"),
		}, {
			name:            "non project owner bypasses interceptor",
			scope:           minder.ObjectOwner_OBJECT_OWNER_USER,
			req:             &request{},
			expectedContext: engine.EntityContext{},
		},
		{
			name: "empty context",
			req: &request{
				Context: &minder.Context{},
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

			if tc.scope == minder.ObjectOwner_OBJECT_OWNER_UNSPECIFIED {
				tc.scope = minder.ObjectOwner_OBJECT_OWNER_PROJECT
			}
			rpcOptions := &minder.RpcOptions{
				AuthScope: tc.scope,
			}

			unaryHandler := func(ctx context.Context, req interface{}) (any, error) {
				return replyType{engine.EntityFromContext(ctx)}, nil
			}
			authorities := auth.UserPermissions{ProjectIds: []uuid.UUID{defaultProjectID}}
			ctx := auth.WithPermissionsContext(withRpcOptions(context.Background(), rpcOptions), authorities)
			reply, err := EntityContextProjectInterceptor(ctx, tc.req, &grpc.UnaryServerInfo{}, unaryHandler)
			if tc.rpcErr != nil {
				assert.Equal(t, tc.rpcErr, err)
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

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
		anonymous bool
		scope     minder.ObjectOwner
		rpcErr    error
	}{
		{
			name:      "anonymous bypasses interceptor",
			entityCtx: &engine.EntityContext{},
			anonymous: true,
		},
		{
			name:      "non project owner bypasses interceptor",
			scope:     minder.ObjectOwner_OBJECT_OWNER_USER,
			entityCtx: &engine.EntityContext{},
		},
		{
			name:      "no permissions error",
			scope:     minder.ObjectOwner_OBJECT_OWNER_PROJECT,
			entityCtx: &engine.EntityContext{},
			rpcErr:    util.UserVisibleError(codes.PermissionDenied, "user is not authorized to access this project"),
		},
		{
			name:  "not authorized on project error",
			scope: minder.ObjectOwner_OBJECT_OWNER_PROJECT,
			entityCtx: &engine.EntityContext{
				Project: engine.Project{
					ID: projectID,
				},
			},
			rpcErr: util.UserVisibleError(codes.PermissionDenied, "user is not authorized to access this project"),
		},
		{
			name:  "authorized on project",
			scope: minder.ObjectOwner_OBJECT_OWNER_PROJECT,
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

			if tc.scope == minder.ObjectOwner_OBJECT_OWNER_UNSPECIFIED {
				tc.scope = minder.ObjectOwner_OBJECT_OWNER_PROJECT
			}
			rpcOptions := &minder.RpcOptions{
				Anonymous: tc.anonymous,
				AuthScope: tc.scope,
			}

			unaryHandler := func(ctx context.Context, req interface{}) (any, error) {
				return replyType{engine.EntityFromContext(ctx)}, nil
			}
			authorities := auth.UserPermissions{ProjectIds: []uuid.UUID{defaultProjectID}}
			ctx := auth.WithPermissionsContext(withRpcOptions(context.Background(), rpcOptions), authorities)
			ctx = engine.WithEntityContext(ctx, tc.entityCtx)
			_, err := ProjectAuthorizationInterceptor(ctx, request{}, &grpc.UnaryServerInfo{}, unaryHandler)
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
