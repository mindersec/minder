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
func TestDataSourceCreateCommand(t *testing.T) {
	fixtureFile := "fixture/mock_datasource_apply.yaml"

	tests := []cli.CmdTestCase{
		{
			Name: "create data source",
			Args: []string{"datasource", "create", "-f", fixtureFile},
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
			GoldenFileName: "datasource_create.table",
		},
		{
			Name: "create grpc error",
			Args: []string{"datasource", "create", "-f", fixtureFile},
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
	}

	cli.RunCmdTests(t, tests, DataSourceCmd)
}
