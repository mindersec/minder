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
func TestDataSourceDeleteCommand(t *testing.T) {
	setupSuccessByName := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

		dataSourceClient.EXPECT().
			DeleteDataSourceByName(gomock.Any(), gomock.Cond(func(r *minderv1.DeleteDataSourceByNameRequest) bool {
				return r.GetName() == "mock-github-rest"
			})).
			Return(&minderv1.DeleteDataSourceByNameResponse{}, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
	}

	setupSuccessById := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

		dataSourceClient.EXPECT().
			DeleteDataSourceById(gomock.Any(), gomock.Cond(func(r *minderv1.DeleteDataSourceByIdRequest) bool {
				return r.GetId() == "00000000-0000-0000-0000-000000000001"
			})).
			Return(&minderv1.DeleteDataSourceByIdResponse{}, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
	}

	tests := []cli.CmdTestCase{
		{
			Name:           "delete data source by name - table output",
			Args:           []string{"datasource", "delete", "-n", "mock-github-rest", "-o", "table"},
			MockSetup:      setupSuccessByName,
			GoldenFileName: "datasource_delete_by_name.table",
		},
		{
			Name:           "delete data source by name - default output (table)",
			Args:           []string{"datasource", "delete", "-n", "mock-github-rest"},
			MockSetup:      setupSuccessByName,
			GoldenFileName: "datasource_delete_by_name.table",
		},
		{
			Name:           "delete data source by name - json output",
			Args:           []string{"datasource", "delete", "-n", "mock-github-rest", "-o", "json"},
			MockSetup:      setupSuccessByName,
			GoldenFileName: "datasource_delete.json",
		},
		{
			Name:           "delete data source by name - yaml output",
			Args:           []string{"datasource", "delete", "-n", "mock-github-rest", "-o", "yaml"},
			MockSetup:      setupSuccessByName,
			GoldenFileName: "datasource_delete.yaml",
		},
		{
			Name:           "delete data source by id - table output",
			Args:           []string{"datasource", "delete", "-i", "00000000-0000-0000-0000-000000000001", "-o", "table"},
			MockSetup:      setupSuccessById,
			GoldenFileName: "datasource_delete_by_id.table",
		},
		{
			Name:           "delete data source by id - json output",
			Args:           []string{"datasource", "delete", "-i", "00000000-0000-0000-0000-000000000001", "-o", "json"},
			MockSetup:      setupSuccessById,
			GoldenFileName: "datasource_delete.json",
		},
		{
			Name:          "fails on invalid output format",
			Args:          []string{"datasource", "delete", "-n", "mock-github-rest", "-o", "invalid_format"},
			ExpectedError: "invalid argument",
		},
		{
			Name:          "fails when neither name nor id provided",
			Args:          []string{"datasource", "delete"},
			ExpectedError: "at least one of the flags in the group [id name] is required",
		},
		{
			Name: "server error handling on delete by name",
			Args: []string{"datasource", "delete", "-n", "mock-github-rest"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

				dataSourceClient.EXPECT().
					DeleteDataSourceByName(gomock.Any(), gomock.Cond(func(r *minderv1.DeleteDataSourceByNameRequest) bool {
						return r.GetName() == "mock-github-rest"
					})).
					Return(nil, status.Error(codes.Internal, "server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
			},
			ExpectedError: "server error",
		},
		{
			Name: "server error handling on delete by id",
			Args: []string{"datasource", "delete", "-i", "00000000-0000-0000-0000-000000000001"},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

				dataSourceClient.EXPECT().
					DeleteDataSourceById(gomock.Any(), gomock.Cond(func(r *minderv1.DeleteDataSourceByIdRequest) bool {
						return r.GetId() == "00000000-0000-0000-0000-000000000001"
					})).
					Return(nil, status.Error(codes.Internal, "server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
			},
			ExpectedError: "server error",
		},
	}

	cli.RunCmdTests(t, tests, DataSourceCmd)
}
