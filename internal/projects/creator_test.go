// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package projects_test

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mockdb "github.com/mindersec/minder/database/mock"
	"github.com/mindersec/minder/internal/auth"
	"github.com/mindersec/minder/internal/auth/jwt"
	authmock "github.com/mindersec/minder/internal/authz/mock"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/marketplaces"
	"github.com/mindersec/minder/internal/projects"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/flags"
)

func TestProvisionSelfEnrolledProject(t *testing.T) {
	t.Parallel()

	authzClient := &authmock.SimpleClient{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().CreateProjectWithID(gomock.Any(), gomock.Any()).
		Return(db.Project{
			ID: uuid.New(),
		}, nil)
	mockStore.EXPECT().CreateEntitlements(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, params db.CreateEntitlementsParams) error {
			expectedFeatures := []string{"featureA", "featureB"}
			if !reflect.DeepEqual(params.Features, expectedFeatures) {
				t.Errorf("expected features %v, got %v", expectedFeatures, params.Features)
			}
			return nil
		})

	ctx := prepareTestToken(context.Background(), t, []any{
		"teamA",
		"teamB",
		"teamC",
	})

	creator := projects.NewProjectCreator(authzClient, marketplaces.NewNoopMarketplace(), &server.DefaultProfilesConfig{}, &server.FeaturesConfig{
		MembershipFeatureMapping: map[string]string{
			"teamA": "featureA",
			"teamB": "featureB",
		},
	}, mockStore, &flags.FakeClient{})

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

	authzClient := &authmock.SimpleClient{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	mockStore.EXPECT().CreateProjectWithID(gomock.Any(), gomock.Any()).
		Return(db.Project{}, fmt.Errorf("failed to create project"))

	ctx := context.Background()
	creator := projects.NewProjectCreator(authzClient, marketplaces.NewNoopMarketplace(), &server.DefaultProfilesConfig{}, &server.FeaturesConfig{}, mockStore, &flags.FakeClient{})
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
		name   string
		errMsg string
	}{
		{"///invalid-name", `name cannot contain '/'`},
		{"", `name cannot be empty`},
		{"longestinvalidnamelongestinvalidnamelongestinvalidnamelongestinvalidnamelongestinvalidname", `name is too long`},
	}

	authzClient := &authmock.SimpleClient{}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mockdb.NewMockStore(ctrl)
	ctx := context.Background()
	creator := projects.NewProjectCreator(authzClient, marketplaces.NewNoopMarketplace(), &server.DefaultProfilesConfig{}, &server.FeaturesConfig{}, mockStore, &flags.FakeClient{})

	for _, tc := range testCases {
		_, err := creator.ProvisionSelfEnrolledProject(
			ctx,
			mockStore,
			tc.name,
			"test-user",
		)
		assert.EqualError(t, err, util.UserVisibleError(codes.InvalidArgument, "invalid project name: validation failed: %s", tc.errMsg).Error())
	}

}

func TestProvisionProject(t *testing.T) {
	t.Parallel()
	parentProject := uuid.New()
	childProj := uuid.New()

	tests := []struct {
		name            string
		setupCtx        func(t *testing.T) context.Context
		setupAuthz      func() *authmock.SimpleClient
		setupStore      func(ctrl *gomock.Controller) *mockdb.MockStore
		featureFlags    flags.Interface
		parentProjectID uuid.UUID
		projectName     string
		wantErrCode     codes.Code
		wantNoErr       bool
	}{
		{
			name: "child project: authzClient.Check denies",
			setupCtx: func(t *testing.T) context.Context {
				t.Helper()
				return prepareTestToken(context.Background(), t, []any{})
			},
			setupAuthz: func() *authmock.SimpleClient {
				return &authmock.SimpleClient{}
			},
			setupStore: func(ctrl *gomock.Controller) *mockdb.MockStore {
				return mockdb.NewMockStore(ctrl)
			},
			featureFlags:    &flags.FakeClient{},
			parentProjectID: parentProject,
			projectName:     "child",
			wantErrCode:     codes.PermissionDenied,
		},
		{
			name: "child project: ProjectAllowsProjectHierarchyOperations returns false",
			setupCtx: func(t *testing.T) context.Context {
				t.Helper()
				return prepareTestToken(context.Background(), t, []any{})
			},
			setupAuthz: func() *authmock.SimpleClient {
				return &authmock.SimpleClient{Allowed: []uuid.UUID{parentProject}}
			},
			setupStore: func(ctrl *gomock.Controller) *mockdb.MockStore {
				s := mockdb.NewMockStore(ctrl)
				s.EXPECT().GetFeatureInProject(gomock.Any(), gomock.Any()).
					Return(nil, sql.ErrNoRows)
				return s
			},
			featureFlags:    &flags.FakeClient{},
			parentProjectID: parentProject,
			projectName:     "child",
			wantErrCode:     codes.PermissionDenied,
		},
		{
			name: "top-level: ProjectCreateDelete flag disabled",
			setupCtx: func(t *testing.T) context.Context {
				t.Helper()
				return prepareTestToken(context.Background(), t, []any{})
			},
			setupAuthz: func() *authmock.SimpleClient {
				return &authmock.SimpleClient{}
			},
			setupStore: func(ctrl *gomock.Controller) *mockdb.MockStore {
				return mockdb.NewMockStore(ctrl)
			},
			featureFlags:    &flags.FakeClient{},
			parentProjectID: uuid.Nil,
			projectName:     "top",
			wantErrCode:     codes.Unimplemented,
		},
		{
			name: "top-level: no identity in context",
			setupCtx: func(_ *testing.T) context.Context {
				return context.Background()
			},
			setupAuthz: func() *authmock.SimpleClient {
				return &authmock.SimpleClient{}
			},
			setupStore: func(ctrl *gomock.Controller) *mockdb.MockStore {
				return mockdb.NewMockStore(ctrl)
			},
			featureFlags: &flags.FakeClient{
				Data: map[string]any{
					string(flags.ProjectCreateDelete): true,
				},
			},
			parentProjectID: uuid.Nil,
			projectName:     "top",
			wantErrCode:     codes.Unauthenticated,
		},
		{
			name: "top-level: happy path returns proto project",
			setupCtx: func(t *testing.T) context.Context {
				t.Helper()
				ctx := prepareTestToken(context.Background(), t, []any{})
				return auth.WithIdentityContext(ctx, &auth.Identity{UserID: "user-1"})
			},
			setupAuthz: func() *authmock.SimpleClient {
				return &authmock.SimpleClient{}
			},
			setupStore: func(ctrl *gomock.Controller) *mockdb.MockStore {
				s := mockdb.NewMockStore(ctrl)
				tx := &sql.Tx{}
				s.EXPECT().BeginTransaction().Return(tx, nil)
				s.EXPECT().GetQuerierWithTransaction(tx).Return(s)
				s.EXPECT().Rollback(tx).Return(nil)
				s.EXPECT().CreateProjectWithID(gomock.Any(), gomock.Any()).
					Return(db.Project{ID: childProj, Name: "top"}, nil)
				s.EXPECT().CreateEntitlements(gomock.Any(), gomock.Any()).Return(nil)
				s.EXPECT().Commit(tx).Return(nil)
				return s
			},
			featureFlags: &flags.FakeClient{
				Data: map[string]any{
					string(flags.ProjectCreateDelete): true,
				},
			},
			parentProjectID: uuid.Nil,
			projectName:     "top",
			wantNoErr:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := tc.setupCtx(t)
			authzClient := tc.setupAuthz()
			mockStore := tc.setupStore(ctrl)

			creator := projects.NewProjectCreator(
				authzClient,
				marketplaces.NewNoopMarketplace(),
				&server.DefaultProfilesConfig{},
				&server.FeaturesConfig{},
				mockStore,
				tc.featureFlags,
			)

			proj, err := creator.ProvisionProject(ctx, tc.projectName, tc.parentProjectID)
			if tc.wantNoErr {
				require.NoError(t, err)
				require.NotNil(t, proj)
				assert.Equal(t, tc.projectName, proj.Name)
			} else {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok, "expected a gRPC status error, got: %v", err)
				assert.Equal(t, tc.wantErrCode, st.Code(),
					"expected code %v, got %v: %v", tc.wantErrCode, st.Code(), err)
				assert.Nil(t, proj)
			}
		})
	}
}

// prepareTestToken creates a JWT token with the specified roles and returns the context with the token.
func prepareTestToken(ctx context.Context, t *testing.T, roles []any) context.Context {
	t.Helper()

	token := openid.New()
	require.NoError(t, token.Set("realm_access", map[string]any{
		"roles": roles,
	}))

	return jwt.WithAuthTokenContext(ctx, token)
}
