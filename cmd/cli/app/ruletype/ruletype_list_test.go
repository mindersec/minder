// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestListCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name: "table output with data",
			Args: []string{"ruletype", "list", "-o", app.Table},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRuleTypeServiceClient(ctrl)
				mockResponse := &minderv1.ListRuleTypesResponse{}
				cli.LoadFixture(t, "mock_ruletypes_response.json", mockResponse)

				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(mockResponse, nil)
				return cli.WithRPCClient[minderv1.RuleTypeServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_populated.table",
		},
		{
			Name: "table output empty",
			Args: []string{"ruletype", "list", "-o", app.Table},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRuleTypeServiceClient(ctrl)
				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(&minderv1.ListRuleTypesResponse{
						RuleTypes: []*minderv1.RuleType{},
					}, nil)
				return cli.WithRPCClient[minderv1.RuleTypeServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_empty.table",
		},
		{
			Name: "yaml output",
			Args: []string{"ruletype", "list", "-o", app.YAML},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRuleTypeServiceClient(ctrl)
				mockResponse := &minderv1.ListRuleTypesResponse{}
				cli.LoadFixture(t, "mock_ruletypes_response.json", mockResponse)

				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(mockResponse, nil)
				return cli.WithRPCClient[minderv1.RuleTypeServiceClient](context.Background(), client)
			},
			GoldenFileName: "list_populated.yaml",
		},
		{
			Name: "grpc error handling",
			Args: []string{"ruletype", "list", "-o", app.Table},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRuleTypeServiceClient(ctrl)
				client.EXPECT().
					ListRuleTypes(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.DeadlineExceeded, "request timed out"))
				return cli.WithRPCClient[minderv1.RuleTypeServiceClient](context.Background(), client)
			},
			ExpectedError: "request timed out",
		},
		{
			Name:          "invalid output format",
			Args:          []string{"ruletype", "list", "-o", "csv"},
			ExpectedError: "invalid argument",
		},
	}

	cli.RunCmdTests(t, tests, ruleTypeCmd)
}
