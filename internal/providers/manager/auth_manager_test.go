// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package manager_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/mindersec/minder/internal/providers/credentials"
	"github.com/mindersec/minder/internal/providers/manager"
	mockmanager "github.com/mindersec/minder/internal/providers/manager/mock"
	"github.com/mindersec/minder/pkg/db"
)

func TestAuthManager_NewAuthManager(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	scenarios := []struct {
		name        string
		oauthMan    *mockmanager.MockproviderClassOAuthManager
		providerMan *mockmanager.MockProviderClassManager
		setupMocks  setupMockCalls
		expectedErr string
	}{
		{
			name: "happy path",
			setupMocks: func(ghClassManager *mockmanager.MockproviderClassOAuthManager, dhClassManager *mockmanager.MockProviderClassManager) {
				ghClassManager.EXPECT().GetSupportedClasses().Return([]db.ProviderClass{db.ProviderClassGithub}).MaxTimes(1)
				dhClassManager.EXPECT().GetSupportedClasses().Return([]db.ProviderClass{db.ProviderClassDockerhub}).MaxTimes(1)
			},
		},
		{
			name: "implementing the same classes",
			setupMocks: func(ghClassManager *mockmanager.MockproviderClassOAuthManager, dhClassManager *mockmanager.MockProviderClassManager) {
				ghClassManager.EXPECT().GetSupportedClasses().Return([]db.ProviderClass{db.ProviderClassGithub}).MaxTimes(1)
				dhClassManager.EXPECT().GetSupportedClasses().Return([]db.ProviderClass{db.ProviderClassGithub}).MaxTimes(1)
			},
			expectedErr: "more than once",
		},
		{
			name: "no registered classes",
			setupMocks: func(ghClassManager *mockmanager.MockproviderClassOAuthManager, _ *mockmanager.MockProviderClassManager) {
				ghClassManager.EXPECT().GetSupportedClasses().Return([]db.ProviderClass{}).MaxTimes(1)
			},
			expectedErr: "no registered classes",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			scenario.oauthMan = mockmanager.NewMockproviderClassOAuthManager(ctrl)
			scenario.providerMan = mockmanager.NewMockProviderClassManager(ctrl)
			scenario.setupMocks(scenario.oauthMan, scenario.providerMan)

			authManager, err := manager.NewAuthManager(scenario.oauthMan, scenario.providerMan)
			if scenario.expectedErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, scenario.expectedErr)
				require.Nil(t, authManager)
			} else {
				require.NoError(t, err)
				require.NotNil(t, authManager)
			}
		})
	}

	dhClassManager := mockmanager.NewMockProviderClassManager(ctrl)
	require.NotNil(t, dhClassManager)
}

func newMockAuthManager(t *testing.T, ctrl *gomock.Controller) (manager.AuthManager, *mockmanager.MockproviderClassOAuthManager, *mockmanager.MockProviderClassManager) {
	t.Helper()

	ghClassManager := mockmanager.NewMockproviderClassOAuthManager(ctrl)
	require.NotNil(t, ghClassManager)
	ghClassManager.EXPECT().GetSupportedClasses().Return([]db.ProviderClass{db.ProviderClassGithub}).MaxTimes(1)

	dhClassManager := mockmanager.NewMockProviderClassManager(ctrl)
	require.NotNil(t, dhClassManager)
	dhClassManager.EXPECT().GetSupportedClasses().Return([]db.ProviderClass{db.ProviderClassDockerhub}).MaxTimes(1)

	authManager, err := manager.NewAuthManager(ghClassManager, dhClassManager)
	require.NoError(t, err)
	require.NotNil(t, authManager)

	return authManager, ghClassManager, dhClassManager
}

type setupMockCalls func(*mockmanager.MockproviderClassOAuthManager, *mockmanager.MockProviderClassManager)

func TestAuthManager_NewOAuthConfig_Validate_ClassManagerProperties(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name          string
		providerClass db.ProviderClass
		setupMocks    setupMockCalls
		expectedErr   string
	}{
		{
			name:          "github implements OAuthManager",
			providerClass: db.ProviderClassGithub,
			setupMocks: func(ghClassManager *mockmanager.MockproviderClassOAuthManager, _ *mockmanager.MockProviderClassManager) {
				ghClassManager.EXPECT().NewOAuthConfig(db.ProviderClassGithub, false).
					Return(&oauth2.Config{
						Endpoint: github.Endpoint,
					}, nil).
					MaxTimes(1)
				ghClassManager.EXPECT().ValidateCredentials(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil).
					MaxTimes(1)
			},
		},
		{
			name:          "dockerhub does not implement OAuthManager",
			providerClass: db.ProviderClassDockerhub,
			setupMocks: func(_ *mockmanager.MockproviderClassOAuthManager, _ *mockmanager.MockProviderClassManager) {
			},
			expectedErr: "class manager does not implement OAuthManager",
		},
		{
			name:          "ghcr is not registered",
			providerClass: db.ProviderClassGhcr,
			setupMocks: func(_ *mockmanager.MockproviderClassOAuthManager, _ *mockmanager.MockProviderClassManager) {
			},
			expectedErr: "error getting class manager",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			authManager, ghClassManager, dhClassManager := newMockAuthManager(t, ctrl)
			scenario.setupMocks(ghClassManager, dhClassManager)

			config, err := authManager.NewOAuthConfig(scenario.providerClass, false)
			if scenario.expectedErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, scenario.expectedErr)
				require.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)
			}

			err = authManager.ValidateCredentials(context.Background(), scenario.providerClass, credentials.NewOAuth2TokenCredential("token"))
			if scenario.expectedErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, scenario.expectedErr)
				require.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)
			}
		})

	}
}

func TestAuthManager_ValidateCredentials(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	authManager, ghClassManager, _ := newMockAuthManager(t, ctrl)

	ghClassManager.EXPECT().ValidateCredentials(
		gomock.Any(),
		credentials.NewGitHubTokenCredential("ghtoken"),
		&manager.CredentialVerifyParams{
			RemoteUser: "remoteuser",
		})

	err := authManager.ValidateCredentials(context.Background(),
		db.ProviderClassGithub,
		credentials.NewGitHubTokenCredential("ghtoken"),
		manager.WithRemoteUser("remoteuser"),
	)
	require.NoError(t, err)
}
