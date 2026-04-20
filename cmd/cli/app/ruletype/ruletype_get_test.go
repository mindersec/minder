// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestGetCommand(t *testing.T) {
	const (
		ruleID   = "00000000-0000-0000-0000-000000000001"
		ruleName = "secret_push_protection"
	)

	tests := []cli.CmdTestCase{
		{
			Name: "get by id - table output",
			Args: []string{"ruletype", "get", "--id", ruleID, "-o", app.Table},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRuleTypeServiceClient(ctrl)
				mockResp := &minderv1.ListRuleTypesResponse{}
				cli.LoadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					GetRuleTypeById(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetRuleTypeByIdResponse{RuleType: mockResp.RuleTypes[0]}, nil)
				return cli.WithRPCClient[minderv1.RuleTypeServiceClient](context.Background(), client)
			},
			GoldenFileName: "get_by_id.table",
		},
		{
			Name: "get by name - yaml output",
			Args: []string{"ruletype", "get", "--name", ruleName, "-o", app.YAML},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRuleTypeServiceClient(ctrl)
				mockResp := &minderv1.ListRuleTypesResponse{}
				cli.LoadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					GetRuleTypeByName(gomock.Any(), gomock.Any()).
					Return(&minderv1.GetRuleTypeByNameResponse{RuleType: mockResp.RuleTypes[0]}, nil)
				return cli.WithRPCClient[minderv1.RuleTypeServiceClient](context.Background(), client)
			},
			GoldenFileName: "get_by_name.yaml",
		},
		{
			Name:          "missing both id and name",
			Args:          []string{"ruletype", "get", "-o", app.Table},
			ExpectedError: "at least one of the flags in the group [id name] is required",
		},
	}

	cli.RunCmdTests(t, tests, ruleTypeCmd)
}
