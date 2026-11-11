// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestRegisterCommand(t *testing.T) {
	const (
		repoName     = "mock-owner/mock-repo"
		alreadyRepo  = "mock-owner/already-registered"
		unregRepo    = "mock-owner/unregistered-repo"
		providerName = "github-app"
	)

	tests := []cli.CmdTestCase{
		{
			Name: "register single repo by name",
			Args: []string{"repo", "register", "-n", repoName, "-p", "github"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRepositoryServiceClient(ctrl)

				mockRemoteResp := &minderv1.ListRemoteRepositoriesFromProviderResponse{}
				cli.LoadFixture(t, "mock_repo_register_remote.json", mockRemoteResp)

				client.EXPECT().
					ListRemoteRepositoriesFromProvider(gomock.Any(), gomock.Any()).
					Return(mockRemoteResp, nil).
					Times(1)

				mockRegisterResp := &minderv1.RegisterRepositoryResponse{}
				cli.LoadFixture(t, "mock_repo_register_success.json", mockRegisterResp)

				client.EXPECT().
					RegisterRepository(gomock.Any(), gomock.Any()).
					Return(mockRegisterResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.RepositoryServiceClient](context.Background(), client)
			},
			GoldenFileName: "register_single.table",
		},
		{
			Name:          "fails when using mutually exclusive flags",
			Args:          []string{"repo", "register", "-n", repoName, "--all"},
			ExpectedError: "cannot use --name and --all together",
		},
		{
			Name:          "fails on invalid repository name",
			Args:          []string{"repo", "register", "-n", "mock-repo"},
			ExpectedError: "invalid repository name",
		},
		{
			Name: "fails when fetch remote repos returns grpc error",
			Args: []string{"repo", "register", "-n", repoName, "-p", "github"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRepositoryServiceClient(ctrl)

				client.EXPECT().
					ListRemoteRepositoriesFromProvider(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "failed to contact upstream provider")).
					Times(1)

				return cli.WithRPCClient[minderv1.RepositoryServiceClient](context.Background(), client)
			},
			ExpectedError: "failed to contact upstream provider",
		},
		{
			Name: "no repos left to register when all remote repos already registered",
			Args: []string{"repo", "register", "-n", repoName, "-p", "github"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRepositoryServiceClient(ctrl)

				client.EXPECT().
					ListRemoteRepositoriesFromProvider(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListRemoteRepositoriesFromProviderResponse{
						Results: []*minderv1.UpstreamRepositoryRef{
							{
								Owner:      "mock-owner",
								Name:       "mock-repo",
								Registered: true,
							},
						},
					}, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.RepositoryServiceClient](context.Background(), client)
			},
			ExpectEmptyStdout: true,
			ExpectedStderr:    "No repos left to register\n",
		},
		{
			Name: "skips already registered repo and registers remaining repo",
			Args: []string{"repo", "register", "-n", alreadyRepo, "-n", unregRepo, "-p", "github"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRepositoryServiceClient(ctrl)

				client.EXPECT().
					ListRemoteRepositoriesFromProvider(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListRemoteRepositoriesFromProviderResponse{
						Results: []*minderv1.UpstreamRepositoryRef{
							{
								Owner:      "mock-owner",
								Name:       "already-registered",
								Registered: true,
							},
							{
								Owner:      "mock-owner",
								Name:       "unregistered-repo",
								Registered: false,
								Context: &minderv1.Context{
									Provider: proto.String("github"),
								},
							},
						},
					}, nil).
					Times(1)

				client.EXPECT().
					RegisterRepository(gomock.Any(), gomock.Cond(func(req *minderv1.RegisterRepositoryRequest) bool {
						return req.GetRepository().GetName() == "unregistered-repo"
					})).
					Return(&minderv1.RegisterRepositoryResponse{
						Result: &minderv1.RegisterRepoResult{
							Repository: &minderv1.Repository{
								Owner: "mock-owner",
								Name:  "unregistered-repo",
							},
							Status: &minderv1.RegisterRepoResult_Status{
								Success: true,
							},
						},
					}, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.RepositoryServiceClient](context.Background(), client)
			},
			ExpectedStderr: "Repository " + alreadyRepo + " is already registered\n",
		},
		{
			Name: "handles registration error with warning and continues",
			Args: []string{"repo", "register", "-n", unregRepo, "-p", "github"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRepositoryServiceClient(ctrl)

				client.EXPECT().
					ListRemoteRepositoriesFromProvider(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListRemoteRepositoriesFromProviderResponse{
						Results: []*minderv1.UpstreamRepositoryRef{
							{
								Owner:      "mock-owner",
								Name:       "unregistered-repo",
								Registered: false,
							},
						},
					}, nil).
					Times(1)

				client.EXPECT().
					RegisterRepository(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.PermissionDenied, "access denied for repo")).
					Times(1)

				return cli.WithRPCClient[minderv1.RepositoryServiceClient](context.Background(), client)
			},
			ExpectEmptyStdout: true,
			ExpectedStderr:    "Error registering repository unregistered-repo: rpc error: code = PermissionDenied desc = access denied for repo\nNo repositories registered\n",
		},
		{
			Name: "enable auto registration for specific provider succeeds",
			Args: []string{"repo", "register", "--all", "-p", providerName},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				provClient := mockv1.NewMockProvidersServiceClient(ctrl)

				configMap := map[string]any{
					"auto_registration": map[string]any{
						"entities": map[string]any{
							minderv1.Entity_ENTITY_REPOSITORIES.ToString(): map[string]any{
								"enabled": false,
							},
						},
					},
				}
				configStruct, err := structpb.NewStruct(configMap)
				if err != nil {
					t.Fatalf("failed to create config struct: %v", err)
				}

				provClient.EXPECT().
					GetProvider(gomock.Any(), gomock.Cond(func(req *minderv1.GetProviderRequest) bool {
						return req.GetName() == providerName
					})).
					Return(&minderv1.GetProviderResponse{
						Provider: &minderv1.Provider{
							Name:   providerName,
							Config: configStruct,
						},
					}, nil).
					Times(1)

				provClient.EXPECT().
					PatchProvider(gomock.Any(), gomock.Cond(func(req *minderv1.PatchProviderRequest) bool {
						return req.GetContext().GetProvider() == providerName
					})).
					Return(&minderv1.PatchProviderResponse{}, nil).
					Times(1)

				provClient.EXPECT().
					ReconcileEntityRegistration(gomock.Any(), gomock.Cond(func(req *minderv1.ReconcileEntityRegistrationRequest) bool {
						return req.GetContext().GetProvider() == providerName && req.GetEntity() == minderv1.Entity_ENTITY_REPOSITORIES.ToString()
					})).
					Return(&minderv1.ReconcileEntityRegistrationResponse{}, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.ProvidersServiceClient](context.Background(), provClient)
			},
			ExpectEmptyStdout: true,
			ExpectedStderr:    "Enabling auto registration succeeded.\nSuccessfully enabled: [" + providerName + "]\nUse `minder repo list` to check the list of registered repositories.\n",
		},
		{
			Name: "enable auto registration for specific provider already enabled",
			Args: []string{"repo", "register", "--all", "-p", providerName},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				provClient := mockv1.NewMockProvidersServiceClient(ctrl)

				configMap := map[string]any{
					"auto_registration": map[string]any{
						"entities": map[string]any{
							minderv1.Entity_ENTITY_REPOSITORIES.ToString(): map[string]any{
								"enabled": true,
							},
						},
					},
				}
				configStruct, err := structpb.NewStruct(configMap)
				if err != nil {
					t.Fatalf("failed to create config struct: %v", err)
				}

				provClient.EXPECT().
					GetProvider(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetProviderResponse{
						Provider: &minderv1.Provider{
							Name:   providerName,
							Config: configStruct,
						},
					}, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.ProvidersServiceClient](context.Background(), provClient)
			},
			ExpectEmptyStdout: true,
			ExpectedStderr:    "Enabling auto registration succeeded.\nAuto registration is already enabled for: [" + providerName + "]\n",
		},
		{
			Name: "enable auto registration across all providers with repo_lister",
			Args: []string{"repo", "register", "--all"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				provClient := mockv1.NewMockProvidersServiceClient(ctrl)

				provClient.EXPECT().
					ListProviders(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListProvidersResponse{
						Providers: []*minderv1.Provider{
							{
								Name:       "non-lister-provider",
								Implements: []minderv1.ProviderType{minderv1.ProviderType_PROVIDER_TYPE_REST},
							},
							{
								Name:       providerName,
								Implements: []minderv1.ProviderType{minderv1.ProviderType_PROVIDER_TYPE_REPO_LISTER},
							},
						},
					}, nil).
					Times(1)

				configStruct, _ := structpb.NewStruct(map[string]any{})
				provClient.EXPECT().
					GetProvider(gomock.Any(), gomock.Cond(func(req *minderv1.GetProviderRequest) bool {
						return req.GetName() == providerName
					})).
					Return(&minderv1.GetProviderResponse{
						Provider: &minderv1.Provider{
							Name:   providerName,
							Config: configStruct,
						},
					}, nil).
					Times(1)

				provClient.EXPECT().
					PatchProvider(gomock.Any(), gomock.Cond(func(req *minderv1.PatchProviderRequest) bool {
						return req.GetContext().GetProvider() == providerName
					})).
					Return(&minderv1.PatchProviderResponse{}, nil).
					Times(1)

				provClient.EXPECT().
					ReconcileEntityRegistration(gomock.Any(), gomock.Any()).
					Return(&minderv1.ReconcileEntityRegistrationResponse{}, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.ProvidersServiceClient](context.Background(), provClient)
			},
			ExpectEmptyStdout: true,
			ExpectedStderr:    "Enabling auto registration succeeded.\nSuccessfully enabled: [" + providerName + "]\nUse `minder repo list` to check the list of registered repositories.\n",
		},
	}

	cli.RunCmdTests(t, tests, RepoCmd)
}
