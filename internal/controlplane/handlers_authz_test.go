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

	"github.com/stacklok/minder/internal/engine"
	minder "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// Mock for HasProtoContext
type request struct {
	Context *minder.Context
}

func (m request) GetContext() *minder.Context {
	return m.Context
}

func TestEntityContextProjectInterceptor(t *testing.T) {
	t.Parallel()
	projectID := uuid.New()
	projectIdStr := projectID.String()
	//nolint:goconst
	provider := "github"

	testCases := []struct {
		name         string
		req          interface{}
		rpcOptions   *minder.RpcOptions
		checkContext func(t *testing.T, ctx context.Context, err error)
	}{
		{
			name: "non project owner bypasses interceptor",
			req:  struct{}{},
			rpcOptions: &minder.RpcOptions{
				Anonymous: false,
				AuthScope: minder.ObjectOwner_OBJECT_OWNER_USER,
			},
			checkContext: func(t *testing.T, ctx context.Context, err error) {
				t.Helper()

				assert.NoError(t, err)

				entity := engine.EntityFromContext(ctx)
				assert.Nil(t, entity)
			},
		},
		{
			name: "sets entity context",
			req: &request{
				Context: &minder.Context{
					Project:  &projectIdStr,
					Provider: &provider,
				},
			},
			rpcOptions: &minder.RpcOptions{
				Anonymous: false,
				AuthScope: minder.ObjectOwner_OBJECT_OWNER_PROJECT,
			},
			checkContext: func(t *testing.T, ctx context.Context, err error) {
				t.Helper()

				assert.NoError(t, err)

				entity := engine.EntityFromContext(ctx)
				assert.Equal(t, projectID, entity.Project.ID)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			unaryHandler := func(ctx context.Context, req interface{}) (any, error) {
				return ctx, nil
			}
			ctx := withRpcOptions(context.Background(), tc.rpcOptions)
			c, err := EntityContextProjectInterceptor(ctx, tc.req, &grpc.UnaryServerInfo{}, unaryHandler)
			ctx, ok := c.(context.Context)
			if !ok {
				t.Errorf("Unexpected error, unary handler should return context: %v", err)
			}

			tc.checkContext(t, ctx, err)
		})
	}
}
