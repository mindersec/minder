// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package datasource

import (
	"context"
	"path/filepath"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestDataSourceApplyCommand(t *testing.T) {
	fixtureFile1 := filepath.Join("fixture", "datasource_test.yaml")
	fixtureFile2 := filepath.Join("fixture", "datasource_test2.yaml")

	setupCreateSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

		var mockResp minderv1.CreateDataSourceResponse
		cli.LoadFixture(t, "mock_datasource_test.json", &mockResp)

		dataSourceClient.EXPECT().
			CreateDataSource(gomock.Any(), gomock.Cond(func(r *minderv1.CreateDataSourceRequest) bool {
				return r.GetDataSource().GetName() == "test-data"
			})).
			Return(&mockResp, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
	}

	setupUpdateSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

		var mockResp minderv1.CreateDataSourceResponse
		cli.LoadFixture(t, "mock_datasource_test.json", &mockResp)

		dataSourceClient.EXPECT().
			CreateDataSource(gomock.Any(), gomock.Cond(func(r *minderv1.CreateDataSourceRequest) bool {
				return r.GetDataSource().GetName() == "test-data"
			})).
			Return(nil, status.Error(codes.AlreadyExists, "data source already exists")).
			Times(1)

		dataSourceClient.EXPECT().
			UpdateDataSource(gomock.Any(), gomock.Cond(func(r *minderv1.UpdateDataSourceRequest) bool {
				return r.GetDataSource().GetName() == "test-data"
			})).
			Return(&minderv1.UpdateDataSourceResponse{DataSource: mockResp.GetDataSource()}, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
	}

	setupMultipleFiles := func(t *testing.T, ctrl *gomock.Controller) context.Context {
		t.Helper()

		dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

		var mockResp1 minderv1.CreateDataSourceResponse
		cli.LoadFixture(t, "mock_datasource_test.json", &mockResp1)

		var mockResp2 minderv1.CreateDataSourceResponse
		cli.LoadFixture(t, "mock_datasource_test2.json", &mockResp2)

		// First file created
		dataSourceClient.EXPECT().
			CreateDataSource(gomock.Any(), gomock.Cond(func(r *minderv1.CreateDataSourceRequest) bool {
				return r.GetDataSource().GetName() == "test-data"
			})).
			Return(&mockResp1, nil).
			Times(1)

		// Second file exists, so falls back to update
		dataSourceClient.EXPECT().
			CreateDataSource(gomock.Any(), gomock.Cond(func(r *minderv1.CreateDataSourceRequest) bool {
				return r.GetDataSource().GetName() == "test-data-2"
			})).
			Return(nil, status.Error(codes.AlreadyExists, "already exists")).
			Times(1)

		dataSourceClient.EXPECT().
			UpdateDataSource(gomock.Any(), gomock.Cond(func(r *minderv1.UpdateDataSourceRequest) bool {
				return r.GetDataSource().GetName() == "test-data-2"
			})).
			Return(&minderv1.UpdateDataSourceResponse{DataSource: mockResp2.GetDataSource()}, nil).
			Times(1)

		return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
	}

	tests := []cli.CmdTestCase{
		{
			Name:              "apply creates new data source via -f flag",
			Args:              []string{"datasource", "apply", "-f", fixtureFile1},
			MockSetup:         setupCreateSuccess,
			GoldenFileName:    "datasource_create.table",
			ExpectEmptyStderr: true,
		},
		{
			Name:              "apply updates existing data source via positional argument",
			Args:              []string{"datasource", "apply", fixtureFile1},
			MockSetup:         setupUpdateSuccess,
			GoldenFileName:    "datasource_create.table",
			ExpectEmptyStderr: true,
		},
		{
			Name:              "apply multiple data sources (one create, one update)",
			Args:              []string{"datasource", "apply", "-f", fixtureFile1, fixtureFile2},
			MockSetup:         setupMultipleFiles,
			GoldenFileName:    "datasource_apply_multiple.table",
			ExpectEmptyStderr: true,
		},
		{
			Name:              "apply fails when no files specified",
			Args:              []string{"datasource", "apply"},
			ExpectedError:     "no files specified: use positional arguments or the -f flag",
			ExpectEmptyStdout: true,
		},
		{
			Name: "apply fails on server error during create",
			Args: []string{"datasource", "apply", "-f", fixtureFile1},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

				dataSourceClient.EXPECT().
					CreateDataSource(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "database connection error")).
					Times(1)

				return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
			},
			ExpectedError:     "database connection error",
			ExpectEmptyStdout: true,
		},
		{
			Name: "apply fails on server error during update",
			Args: []string{"datasource", "apply", "-f", fixtureFile1},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

				dataSourceClient.EXPECT().
					CreateDataSource(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.AlreadyExists, "already exists")).
					Times(1)

				dataSourceClient.EXPECT().
					UpdateDataSource(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "failed to update data source")).
					Times(1)

				return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
			},
			ExpectedError:     "failed to update data source",
			ExpectEmptyStdout: true,
		},
	}

	cli.RunCmdTests(t, tests, DataSourceCmd)
}
