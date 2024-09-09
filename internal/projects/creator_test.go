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

package projects_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	mockdb "github.com/stacklok/minder/database/mock"
	"github.com/stacklok/minder/internal/authz/mock"
	"github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/marketplaces"
	"github.com/stacklok/minder/internal/projects"
)

func TestProvisionSelfEnrolledProject(t *testing.T) {
	t.Parallel()

	authzClient := &mock.SimpleClient{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().CreateProjectWithID(gomock.Any(), gomock.Any()).
		Return(db.Project{
			ID: uuid.New(),
		}, nil)

	ctx := context.Background()

	creator := projects.NewProjectCreator(authzClient, marketplaces.NewNoopMarketplace(), &server.DefaultProfilesConfig{})
	_, err := creator.ProvisionSelfEnrolledProject(
		ctx,
		mockStore,
		"test-proj",
		"test-user",
	)
	assert.NoError(t, err)

	t.Log("ensure project permission was written")
	assert.Len(t, authzClient.Allowed, 1)
}

func TestProvisionSelfEnrolledProjectFailsWritingProjectToDB(t *testing.T) {
	t.Parallel()

	authzClient := &mock.SimpleClient{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().CreateProjectWithID(gomock.Any(), gomock.Any()).
		Return(db.Project{}, fmt.Errorf("failed to create project"))

	ctx := context.Background()

	creator := projects.NewProjectCreator(authzClient, marketplaces.NewNoopMarketplace(), &server.DefaultProfilesConfig{})
	_, err := creator.ProvisionSelfEnrolledProject(
		ctx,
		mockStore,
		"test-proj",
		"test-user",
	)
	assert.Error(t, err)

	t.Log("ensure project permission was cleaned up")
	assert.Len(t, authzClient.Allowed, 0)
}

func TestProvisionSelfEnrolledProjectInvalidName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
	}{
		{"///invalid-name"},
		{""},
		{"longestinvalidnamelongestinvalidnamelongestinvalidnamelongestinvalidnamelongestinvalidname"},
	}

	authzClient := &mock.SimpleClient{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	ctx := context.Background()
	creator := projects.NewProjectCreator(authzClient, marketplaces.NewNoopMarketplace(), &server.DefaultProfilesConfig{})

	for _, tc := range testCases {
		_, err := creator.ProvisionSelfEnrolledProject(
			ctx,
			mockStore,
			tc.name,
			"test-user",
		)
		assert.EqualError(t, err, "invalid project name: validation failed")
	}

}
