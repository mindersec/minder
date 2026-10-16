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
func TestDataSourceListCommand(t *testing.T) {
	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

		mockResp := &minderv1.ListDataSourcesResponse{}
		cli.LoadFixture(t, "mock_datasource_list.json", mockResp)

		dataSourceClient.EXPECT().
			ListDataSources(gomock.Any(), gomock.Any()).
			Return(mockResp, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "list data sources - table output",
			Args:           []string{"datasource", "list", "-o", "table"},
			MockSetup:      setupSuccess,
			GoldenFileName: "datasource_list.table",
		},
		{
			Name:           "list data sources - default output (table)",
			Args:           []string{"datasource", "list"},
			MockSetup:      setupSuccess,
			GoldenFileName: "datasource_list.table",
		},
		{
			Name:           "list data sources - json output",
			Args:           []string{"datasource", "list", "-o", "json"},
			MockSetup:      setupSuccess,
			GoldenFileName: "datasource_list.json",
		},
		{
			Name:           "list data sources - yaml output",
			Args:           []string{"datasource", "list", "-o", "yaml"},
			MockSetup:      setupSuccess,
			GoldenFileName: "datasource_list.yaml",
		},
		{
			Name: "list data sources - empty table output",
			Args: []string{"datasource", "list", "-o", "table"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

				emptyResp := &minderv1.ListDataSourcesResponse{}
				cli.LoadFixture(t, "mock_datasource_list_empty.json", emptyResp)

				dataSourceClient.EXPECT().
					ListDataSources(gomock.Any(), gomock.Any()).
					Return(emptyResp, nil).
					Times(1)

				return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
			},
			GoldenFileName: "datasource_list_empty.table",
		},
		{
			Name:          "fails on invalid output format",
			Args:          []string{"datasource", "list", "-o", "invalid_format"},
			ExpectedError: "invalid argument",
		},
		{
			Name: "server error handling",
			Args: []string{"datasource", "list"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

				dataSourceClient.EXPECT().
					ListDataSources(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
			},
			ExpectedError: "server error",
		},
	}

	cli.RunCmdTests(t, tests, DataSourceCmd)
}
