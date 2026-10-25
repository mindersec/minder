// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest
func TestListProviderCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name: "list populated",
			Args: []string{"provider", "list"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProvidersServiceClient(ctrl)
				mockResponse := &minderv1.ListProvidersResponse{}
				cli.LoadFixture(t, "mock_provider_response.json", mockResponse)
				client.EXPECT().
					ListProviders(gomock.Any(), gomock.Any()).
					Return(mockResponse, nil)
				return cli.WithRPCClient[minderv1.ProvidersServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_populated.table",
		},
		{
			Name: "list empty",
			Args: []string{"provider", "list"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProvidersServiceClient(ctrl)
				client.EXPECT().
					ListProviders(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListProvidersResponse{
						Providers: []*minderv1.Provider{},
					}, nil)
				return cli.WithRPCClient[minderv1.ProvidersServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_empty.table",
		},
		{
			Name: "list grpc error",
			Args: []string{"provider", "list"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProvidersServiceClient(ctrl)
				client.EXPECT().
					ListProviders(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.DeadlineExceeded, "request timed out"))
				return cli.WithRPCClient[minderv1.ProvidersServiceClient](context.Background(), client)
			},
			ExpectedError: "request timed out",
		},
	}

	cli.RunCmdTests(t, tests, ProviderCmd)
}

//nolint:paralleltest
func TestDeleteProviderCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name: "delete by name",
			Args: []string{"provider", "delete", "-n", "github-app"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProvidersServiceClient(ctrl)
				client.EXPECT().
					DeleteProvider(gomock.Any(), gomock.Cond(func(req any) bool {
						r, ok := req.(*minderv1.DeleteProviderRequest)
						return ok && r.GetContext().GetProvider() == "github-app"
					})).
					Return(&minderv1.DeleteProviderResponse{
						Name: "github-app",
					}, nil)
				return cli.WithRPCClient[minderv1.ProvidersServiceClient](context.Background(), client)
			},
			ExpectedStderr:    "Successfully deleted provider with name: github-app\n",
			ExpectEmptyStdout: true,
		},
		{
			Name: "delete by id",
			Args: []string{"provider", "delete", "-i", "test-id"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProvidersServiceClient(ctrl)
				client.EXPECT().
					DeleteProviderByID(gomock.Any(), gomock.Cond(func(req any) bool {
						r, ok := req.(*minderv1.DeleteProviderByIDRequest)
						return ok && r.GetId() == "test-id"
					})).
					Return(&minderv1.DeleteProviderByIDResponse{
						Id: "test-id",
					}, nil)
				return cli.WithRPCClient[minderv1.ProvidersServiceClient](context.Background(), client)
			},
			ExpectedStderr:    "Successfully deleted provider with id: test-id\n",
			ExpectEmptyStdout: true,
		},
		{
			Name: "delete grpc error",
			Args: []string{"provider", "delete", "-n", "github-app"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProvidersServiceClient(ctrl)
				client.EXPECT().
					DeleteProvider(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "provider not found"))
				return cli.WithRPCClient[minderv1.ProvidersServiceClient](context.Background(), client)
			},
			ExpectedError:     "provider not found",
			ExpectEmptyStdout: true,
		},
		{
			Name:          "fails when missing both name and id",
			Args:          []string{"provider", "delete"},
			ExpectedError: `at least one of the flags in the group [name id] is required`,
		},
		{
			Name:          "fails when providing both name and id",
			Args:          []string{"provider", "delete", "-n", "github-app", "-i", "test-id"},
			ExpectedError: `none of the others can be`,
		},
	}

	cli.RunCmdTests(t, tests, ProviderCmd)
}

//nolint:paralleltest
func TestGetProviderCommand(t *testing.T) {
	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()
		client := mockv1.NewMockProvidersServiceClient(ctrl)
		mockResponse := &minderv1.GetProviderResponse{}
		cli.LoadFixture(t, "mock_provider_get_response.json", mockResponse)
		client.EXPECT().
			GetProvider(gomock.Any(), gomock.Cond(func(req any) bool {
				r, ok := req.(*minderv1.GetProviderRequest)
				return ok && r.GetName() == "github-app"
			})).
			Return(mockResponse, nil)
		return cli.WithRPCClient[minderv1.ProvidersServiceClient](context.Background(), client)
	}

	tests := []cli.CmdTestCase{
		{
			Name:              "get provider - table output",
			Args:              []string{"provider", "get", "-n", "github-app", "-o", "table"},
			MockSetup:         setupSuccess,
			GoldenFileName:    "get_provider.table",
			ExpectEmptyStderr: true,
		},
		{
			Name:              "get provider - default output (table)",
			Args:              []string{"provider", "get", "-n", "github-app"},
			MockSetup:         setupSuccess,
			GoldenFileName:    "get_provider.table",
			ExpectEmptyStderr: true,
		},
		{
			Name:              "get provider - json output",
			Args:              []string{"provider", "get", "-n", "github-app", "-o", "json"},
			MockSetup:         setupSuccess,
			GoldenFileName:    "get_provider.json",
			ExpectEmptyStderr: true,
		},
		{
			Name:              "get provider - yaml output",
			Args:              []string{"provider", "get", "-n", "github-app", "-o", "yaml"},
			MockSetup:         setupSuccess,
			GoldenFileName:    "get_provider.yaml",
			ExpectEmptyStderr: true,
		},
		{
			Name: "get grpc error",
			Args: []string{"provider", "get", "-n", "github-app"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProvidersServiceClient(ctrl)
				client.EXPECT().
					GetProvider(gomock.Any(), gomock.Cond(func(req any) bool {
						r, ok := req.(*minderv1.GetProviderRequest)
						return ok && r.GetName() == "github-app"
					})).
					Return(nil, status.Error(codes.NotFound, "provider not found"))
				return cli.WithRPCClient[minderv1.ProvidersServiceClient](context.Background(), client)
			},
			ExpectedError:     "provider not found",
			ExpectEmptyStdout: true,
		},
		{
			Name:          "fails when name flag is missing",
			Args:          []string{"provider", "get"},
			ExpectedError: `required flag(s) "name" not set`,
		},
		{
			Name:          "fails on invalid output format",
			Args:          []string{"provider", "get", "-n", "github-app", "-o", "invalid_format"},
			ExpectedError: "invalid argument",
		},
	}

	cli.RunCmdTests(t, tests, ProviderCmd)
}
