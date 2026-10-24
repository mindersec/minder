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

//nolint:paralleltest
func TestDataSourceApplyCommand(t *testing.T) {
	fixtureFile := "fixture/mock_datasource_apply.yaml"

	tests := []cli.CmdTestCase{
		{
			Name: "apply creates new data source",
			Args: []string{"datasource", "apply", "-f", fixtureFile},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockDataSourceServiceClient(ctrl)
				mockResp := &minderv1.CreateDataSourceResponse{}
				cli.LoadFixture(t, "mock_datasource_create.json", mockResp)
				client.EXPECT().
					CreateDataSource(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)
				return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), client)
			},
			GoldenFileName: "datasource_apply_create.table",
		},
		{
			Name: "apply updates existing data source",
			Args: []string{"datasource", "apply", "-f", fixtureFile},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockDataSourceServiceClient(ctrl)
				mockResp := &minderv1.UpdateDataSourceResponse{}
				cli.LoadFixture(t, "mock_datasource_update.json", mockResp)
				client.EXPECT().
					CreateDataSource(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.AlreadyExists, "already exists"))
				client.EXPECT().
					UpdateDataSource(gomock.Any(), gomock.Any()).
					Return(mockResp, nil)
				return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), client)
			},
			GoldenFileName: "datasource_apply_update.table",
		},
		{
			Name: "apply grpc error",
			Args: []string{"datasource", "apply", "-f", fixtureFile},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockDataSourceServiceClient(ctrl)
				client.EXPECT().
					CreateDataSource(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.Internal, "server error"))
				return cli.WithRPCClient[minderv1.DataSourceServiceClient](context.Background(), client)
			},
			ExpectedError: "server error",
		},
		{
			Name:          "apply no files specified",
			Args:          []string{"datasource", "apply"},
			ExpectedError: "no files specified",
		},
	}

	cli.RunCmdTests(t, tests, DataSourceCmd)
}
