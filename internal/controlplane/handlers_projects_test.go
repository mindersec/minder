// Copyright 2024 Stacklok, Inc
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
	"testing"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/auth/jwt"
	"github.com/mindersec/minder/internal/authz/mock"
	"github.com/mindersec/minder/internal/db"
	minder "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestListProjects(t *testing.T) {
	t.Parallel()

	user := openid.New()
	assert.NoError(t, user.Set("sub", "testuser"))

	authzClient := &mock.SimpleClient{
		Allowed: []uuid.UUID{uuid.New()},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().GetUserBySubject(gomock.Any(), user.Subject()).Return(db.User{ID: 1}, nil)
	mockStore.EXPECT().GetProjectByID(gomock.Any(), authzClient.Allowed[0]).Return(
		db.Project{ID: authzClient.Allowed[0]}, nil)

	server := Server{
		store:       mockStore,
		authzClient: authzClient,
	}

	ctx := context.Background()
	ctx = jwt.WithAuthTokenContext(ctx, user)

	resp, err := server.ListProjects(ctx, &minder.ListProjectsRequest{})
	assert.NoError(t, err)

	assert.Len(t, resp.Projects, 1)
	assert.Equal(t, authzClient.Allowed[0].String(), resp.Projects[0].ProjectId)
}

func TestListProjectsWithOneDeletedWhileIterating(t *testing.T) {
	t.Parallel()

	user := openid.New()
	assert.NoError(t, user.Set("sub", "testuser"))

	authzClient := &mock.SimpleClient{
		Allowed: []uuid.UUID{uuid.New(), uuid.New(), uuid.New()},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().GetUserBySubject(gomock.Any(), user.Subject()).Return(db.User{ID: 1}, nil)
	mockStore.EXPECT().GetProjectByID(gomock.Any(), authzClient.Allowed[0]).Return(
		db.Project{ID: authzClient.Allowed[0]}, nil)
	mockStore.EXPECT().GetProjectByID(gomock.Any(), authzClient.Allowed[1]).Return(
		db.Project{}, sql.ErrNoRows)
	mockStore.EXPECT().GetProjectByID(gomock.Any(), authzClient.Allowed[2]).Return(
		db.Project{ID: authzClient.Allowed[2]}, nil)

	server := Server{
		store:       mockStore,
		authzClient: authzClient,
	}

	ctx := context.Background()
	ctx = jwt.WithAuthTokenContext(ctx, user)

	resp, err := server.ListProjects(ctx, &minder.ListProjectsRequest{})
	assert.NoError(t, err)

	assert.Len(t, resp.Projects, 2)
	assert.Equal(t, authzClient.Allowed[0].String(), resp.Projects[0].ProjectId)
	assert.Equal(t, authzClient.Allowed[2].String(), resp.Projects[1].ProjectId)
}
