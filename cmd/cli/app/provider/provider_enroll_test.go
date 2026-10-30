// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

func TestProviderConfigFromArg(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for empty config source", func(t *testing.T) {
		t.Parallel()

		cfg, err := providerConfigFromArg("", nil)
		require.NoError(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("parses valid JSON from stdin (-)", func(t *testing.T) {
		t.Parallel()

		jsonInput := `{"endpoint": "https://gitlab.example.com/api/v4", "name": "my-gitlab"}`
		reader := strings.NewReader(jsonInput)

		cfg, err := providerConfigFromArg("-", reader)
		require.NoError(t, err)
		require.NotNil(t, cfg)

		fields := cfg.GetFields()
		assert.Equal(t, "https://gitlab.example.com/api/v4", fields["endpoint"].GetStringValue())
		assert.Equal(t, "my-gitlab", fields["name"].GetStringValue())
	})

	t.Run("fails on malformed JSON from stdin", func(t *testing.T) {
		t.Parallel()

		malformedJSON := `{"endpoint": "incomplete`
		reader := strings.NewReader(malformedJSON)

		_, err := providerConfigFromArg("-", reader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error parsing provider configuration")
	})

	t.Run("fails when file does not exist", func(t *testing.T) {
		t.Parallel()

		_, err := providerConfigFromArg("non_existent_provider_config.json", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error opening file arg")
	})
}

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestEnrollProviderCmd(t *testing.T) {
	const zeroUUID = "00000000-0000-0000-0000-000000000000"

	tests := []cli.CmdTestCase{
		{
			Name: "enroll using token - successful creation and token storage",
			Args: []string{
				"provider", "enroll",
				"-p", "gitlab",
				"--token", "glpat-testtoken123",
				"--owner", "my-org",
			},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				provClient := mockv1.NewMockProvidersServiceClient(ctrl)
				oauthClient := mockv1.NewMockOAuthServiceClient(ctrl)

				provClient.EXPECT().
					ListProviderClasses(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListProviderClassesResponse{
						ProviderClassInfos: []*minderv1.ProviderClassInfo{
							{
								Class: "gitlab",
								SupportedAuthFlows: []minderv1.AuthorizationFlow{
									minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT,
									minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_OAUTH2_AUTHORIZATION_CODE_FLOW,
								},
							},
						},
					}, nil)

				provClient.EXPECT().
					CreateProvider(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateProviderResponse{}, nil)

				oauthClient.EXPECT().
					StoreProviderToken(gomock.Any(), gomock.Any()).
					Return(&minderv1.StoreProviderTokenResponse{}, nil)

				ctx := context.Background()
				ctx = cli.WithRPCClient[minderv1.ProvidersServiceClient](ctx, provClient)
				ctx = cli.WithRPCClient[minderv1.OAuthServiceClient](ctx, oauthClient)
				return ctx
			},
			ExpectedStderr: "Provider enrolled successfully\n",
		},
		{
			Name: "enroll using token - provider already exists updates token",
			Args: []string{
				"provider", "enroll",
				"-p", "gitlab",
				"--token", "glpat-testtoken123",
			},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				provClient := mockv1.NewMockProvidersServiceClient(ctrl)
				oauthClient := mockv1.NewMockOAuthServiceClient(ctrl)

				provClient.EXPECT().
					ListProviderClasses(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListProviderClassesResponse{
						ProviderClassInfos: []*minderv1.ProviderClassInfo{
							{
								Class: "gitlab",
								SupportedAuthFlows: []minderv1.AuthorizationFlow{
									minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT,
								},
							},
						},
					}, nil)

				provClient.EXPECT().
					CreateProvider(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.AlreadyExists, "provider already exists"))

				oauthClient.EXPECT().
					StoreProviderToken(gomock.Any(), gomock.Any()).
					Return(&minderv1.StoreProviderTokenResponse{}, nil)

				ctx := context.Background()
				ctx = cli.WithRPCClient[minderv1.ProvidersServiceClient](ctx, provClient)
				ctx = cli.WithRPCClient[minderv1.OAuthServiceClient](ctx, oauthClient)
				return ctx
			},
			ExpectedStderr: "Provider enrolled successfully\n",
		},
		{
			Name: "enroll using token - create provider fails with internal error",
			Args: []string{
				"provider", "enroll",
				"-p", "gitlab",
				"--token", "glpat-testtoken123",
			},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				provClient := mockv1.NewMockProvidersServiceClient(ctrl)
				oauthClient := mockv1.NewMockOAuthServiceClient(ctrl)

				provClient.EXPECT().
					ListProviderClasses(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListProviderClassesResponse{
						ProviderClassInfos: []*minderv1.ProviderClassInfo{
							{
								Class: "gitlab",
								SupportedAuthFlows: []minderv1.AuthorizationFlow{
									minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT,
								},
							},
						},
					}, nil)

				provClient.EXPECT().
					CreateProvider(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "internal server error"))

				ctx := context.Background()
				ctx = cli.WithRPCClient[minderv1.ProvidersServiceClient](ctx, provClient)
				ctx = cli.WithRPCClient[minderv1.OAuthServiceClient](ctx, oauthClient)
				return ctx
			},
			ExpectedError: "internal server error",
		},
		{
			Name: "enroll using token - store token fails",
			Args: []string{
				"provider", "enroll",
				"-p", "gitlab",
				"--token", "glpat-testtoken123",
			},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				provClient := mockv1.NewMockProvidersServiceClient(ctrl)
				oauthClient := mockv1.NewMockOAuthServiceClient(ctrl)

				provClient.EXPECT().
					ListProviderClasses(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListProviderClassesResponse{
						ProviderClassInfos: []*minderv1.ProviderClassInfo{
							{
								Class: "gitlab",
								SupportedAuthFlows: []minderv1.AuthorizationFlow{
									minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT,
								},
							},
						},
					}, nil)

				provClient.EXPECT().
					CreateProvider(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateProviderResponse{}, nil)

				oauthClient.EXPECT().
					StoreProviderToken(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.InvalidArgument, "invalid token"))

				ctx := context.Background()
				ctx = cli.WithRPCClient[minderv1.ProvidersServiceClient](ctx, provClient)
				ctx = cli.WithRPCClient[minderv1.OAuthServiceClient](ctx, oauthClient)
				return ctx
			},
			ExpectedError: "invalid token",
		},
		{
			Name: "enroll with unknown provider class",
			Args: []string{
				"provider", "enroll",
				"-p", "unknown-class",
				"--token", "token123",
			},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				provClient := mockv1.NewMockProvidersServiceClient(ctrl)
				oauthClient := mockv1.NewMockOAuthServiceClient(ctrl)

				provClient.EXPECT().
					ListProviderClasses(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListProviderClassesResponse{
						ProviderClassInfos: []*minderv1.ProviderClassInfo{
							{
								Class: "github",
							},
						},
					}, nil)

				ctx := context.Background()
				ctx = cli.WithRPCClient[minderv1.ProvidersServiceClient](ctx, provClient)
				ctx = cli.WithRPCClient[minderv1.OAuthServiceClient](ctx, oauthClient)
				return ctx
			},
			ExpectedError: "provider class unknown-class not supported",
		},
		{
			Name: "enroll with missing config file",
			Args: []string{
				"provider", "enroll",
				"-p", "gitlab",
				"--provider-config", filepath.Join("testdata", "non_existent.json"),
			},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				provClient := mockv1.NewMockProvidersServiceClient(ctrl)
				oauthClient := mockv1.NewMockOAuthServiceClient(ctrl)

				ctx := context.Background()
				ctx = cli.WithRPCClient[minderv1.ProvidersServiceClient](ctx, provClient)
				ctx = cli.WithRPCClient[minderv1.OAuthServiceClient](ctx, oauthClient)
				return ctx
			},
			ExpectedError: "error opening file arg",
		},
		{
			Name: "enroll fails when legacy github provider already enrolled",
			Args: []string{
				"provider", "enroll",
				"-p", "github-app",
				"--name", "my-new-app",
				"--skip-browser",
			},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				provClient := mockv1.NewMockProvidersServiceClient(ctrl)
				oauthClient := mockv1.NewMockOAuthServiceClient(ctrl)

				provClient.EXPECT().
					ListProviderClasses(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListProviderClassesResponse{
						ProviderClassInfos: []*minderv1.ProviderClassInfo{
							{
								Class: "github-app",
								SupportedAuthFlows: []minderv1.AuthorizationFlow{
									minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_OAUTH2_AUTHORIZATION_CODE_FLOW,
								},
							},
						},
					}, nil)

				provClient.EXPECT().
					ListProviders(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListProvidersResponse{
						Providers: []*minderv1.Provider{
							{
								Name:             "github",
								Class:            legacyGitHubProvider.ToString(),
								CredentialsState: minderv1.CredentialsState_CREDENTIALS_STATE_SET.ToString(),
							},
						},
					}, nil)

				ctx := context.Background()
				ctx = cli.WithRPCClient[minderv1.ProvidersServiceClient](ctx, provClient)
				ctx = cli.WithRPCClient[minderv1.OAuthServiceClient](ctx, oauthClient)
				return ctx
			},
			ExpectedError: "it seems you are using the legacy github provider",
		},
	}

	_ = zeroUUID
	cli.RunCmdTests(t, tests, enrollCmd)
}
