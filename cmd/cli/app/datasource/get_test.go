// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package datasource

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

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestDataSourceGetCommand(t *testing.T) {
	setupSuccessByName := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

		mockResp := &minderv1.GetDataSourceByNameResponse{}
		cli.LoadFixture(t, "mock_datasource_get.json", mockResp)

		dataSourceClient.EXPECT().
			GetDataSourceByName(gomock.Any(), gomock.Any()).
			Return(mockResp, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
	}

	setupSuccessStructuredByName := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

		mockResp := &minderv1.GetDataSourceByNameResponse{}
		cli.LoadFixture(t, "mock_datasource_get_structured.json", mockResp)

		dataSourceClient.EXPECT().
			GetDataSourceByName(gomock.Any(), gomock.Any()).
			Return(mockResp, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
	}

	setupSuccessById := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

		mockResp := &minderv1.GetDataSourceByIdResponse{}
		cli.LoadFixture(t, "mock_datasource_get.json", mockResp)

		dataSourceClient.EXPECT().
			GetDataSourceById(gomock.Any(), gomock.Any()).
			Return(mockResp, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "get data source by name - table output",
			Args:           []string{"datasource", "get", "-n", "mock-github-rest", "-o", "table"},
			MockSetup:      setupSuccessByName,
			GoldenFileName: "datasource_get.table",
		},
		{
			Name:           "get data source by name - default output (table)",
			Args:           []string{"datasource", "get", "-n", "mock-github-rest"},
			MockSetup:      setupSuccessByName,
			GoldenFileName: "datasource_get.table",
		},
		{
			Name:           "get data source by name - json output",
			Args:           []string{"datasource", "get", "-n", "mock-github-rest", "-o", "json"},
			MockSetup:      setupSuccessByName,
			GoldenFileName: "datasource_get.json",
		},
		{
			Name:           "get data source by name - yaml output",
			Args:           []string{"datasource", "get", "-n", "mock-github-rest", "-o", "yaml"},
			MockSetup:      setupSuccessByName,
			GoldenFileName: "datasource_get.yaml",
		},
		{
			Name:           "get data source by name with structured driver - table output",
			Args:           []string{"datasource", "get", "-n", "mock-security-docs", "-o", "table"},
			MockSetup:      setupSuccessStructuredByName,
			GoldenFileName: "datasource_get_structured.table",
		},
		{
			Name:           "get data source by id - table output",
			Args:           []string{"datasource", "get", "-i", "00000000-0000-0000-0000-000000000001", "-o", "table"},
			MockSetup:      setupSuccessById,
			GoldenFileName: "datasource_get.table",
		},
		{
			Name:          "fails on invalid output format",
			Args:          []string{"datasource", "get", "-n", "mock-github-rest", "-o", "invalid_format"},
			ExpectedError: "invalid argument",
		},
		{
			Name:          "fails when neither name nor id provided",
			Args:          []string{"datasource", "get"},
			ExpectedError: "at least one of the flags in the group [id name] is required",
		},
		{
			Name: "server error handling on get by name",
			Args: []string{"datasource", "get", "-n", "mock-github-rest"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

				dataSourceClient.EXPECT().
					GetDataSourceByName(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
			},
			ExpectedError: "server error",
		},
		{
			Name: "server error handling on get by id",
			Args: []string{"datasource", "get", "-i", "00000000-0000-0000-0000-000000000001"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

				dataSourceClient.EXPECT().
					GetDataSourceById(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
			},
			ExpectedError: "server error",
		},
	}

	cli.RunCmdTests(t, tests, DataSourceCmd)
}
