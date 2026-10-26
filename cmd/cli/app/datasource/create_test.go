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
func TestDataSourceCreateCommand(t *testing.T) {
	fixtureFile := filepath.Join("fixture", "datasource_test.yaml")

	setupSuccess := func(t *testing.T, ctrl *gomock.Controller) context.Context {
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

	tests := []cli.CmdTestCase{
		{
			Name:              "create data source - success",
			Args:              []string{"datasource", "create", "-f", fixtureFile},
			MockSetup:         setupSuccess,
			GoldenFileName:    "datasource_create.table",
			ExpectEmptyStderr: true,
		},
		{
			Name: "create data source - failure on server error",
			Args: []string{"datasource", "create", "-f", fixtureFile},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()

				dataSourceClient := mockv1.NewMockDataSourceServiceClient(ctrl)

				dataSourceClient.EXPECT().
					CreateDataSource(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "internal server error")).
					Times(1)

				return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), dataSourceClient)
			},
			ExpectedError:     "internal server error",
			ExpectEmptyStdout: true,
		},
		{
			Name:              "create data source - missing file flag",
			Args:              []string{"datasource", "create"},
			ExpectedError:     "required flag(s) \"file\" not set",
			ExpectEmptyStdout: true,
		},
		{
			Name:              "create data source - non-existent file",
			Args:              []string{"datasource", "create", "-f", "non_existent_file.yaml"},
			ExpectedError:     "error getting file info",
			ExpectEmptyStdout: true,
		},
	}

	cli.RunCmdTests(t, tests, DataSourceCmd)
}
