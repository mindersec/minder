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
					DeleteProvider(gomock.Any(), gomock.Any()).
					Return(&minderv1.DeleteProviderResponse{
						Name: "github-app",
					}, nil)
				return cli.WithRPCClient[minderv1.ProvidersServiceClient](context.Background(), client)
			},
			GoldenFileName: "delete_by_name.txt",
		},
		{
			Name: "delete by id",
			Args: []string{"provider", "delete", "-i", "test-id"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProvidersServiceClient(ctrl)
				client.EXPECT().
					DeleteProviderByID(gomock.Any(), gomock.Any()).
					Return(&minderv1.DeleteProviderByIDResponse{
						Id: "test-id",
					}, nil)
				return cli.WithRPCClient[minderv1.ProvidersServiceClient](context.Background(), client)
			},
			GoldenFileName: "delete_by_id.txt",
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
			ExpectedError: "provider not found",
		},
	}

	cli.RunCmdTests(t, tests, ProviderCmd)
}

//nolint:paralleltest
func TestGetProviderCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name: "get grpc error",
			Args: []string{"provider", "get", "-n", "github-app"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockProvidersServiceClient(ctrl)
				client.EXPECT().
					GetProvider(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.NotFound, "provider not found"))
				return cli.WithRPCClient[minderv1.ProvidersServiceClient](context.Background(), client)
			},
			ExpectedError: "provider not found",
		},
	}

	cli.RunCmdTests(t, tests, ProviderCmd)
}
