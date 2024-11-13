// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package projects_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/auth/jwt"
	"github.com/mindersec/minder/internal/authz/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/marketplaces"
	"github.com/mindersec/minder/internal/projects"
	"github.com/mindersec/minder/pkg/config/server"
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
	keyCloakUserToken := openid.New()
	require.NoError(t, keyCloakUserToken.Set("realm_access", map[string]interface{}{
		"roles": []interface{}{
			"default-roles-stacklok",
			"offline_access",
			"uma_authorization",
		},
	}))
	ctx = jwt.WithAuthTokenContext(ctx, keyCloakUserToken)

	creator := projects.NewProjectCreator(authzClient, marketplaces.NewNoopMarketplace(), &server.DefaultProfilesConfig{}, &server.FeaturesConfig{})
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
	keyCloakUserToken := openid.New()
	require.NoError(t, keyCloakUserToken.Set("realm_access", map[string]interface{}{
		"roles": []interface{}{
			"default-roles-stacklok",
			"offline_access",
			"uma_authorization",
		},
	}))
	ctx = jwt.WithAuthTokenContext(ctx, keyCloakUserToken)

	creator := projects.NewProjectCreator(authzClient, marketplaces.NewNoopMarketplace(), &server.DefaultProfilesConfig{}, &server.FeaturesConfig{})
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
	creator := projects.NewProjectCreator(authzClient, marketplaces.NewNoopMarketplace(), &server.DefaultProfilesConfig{}, &server.FeaturesConfig{})

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
