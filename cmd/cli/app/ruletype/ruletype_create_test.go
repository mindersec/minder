// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"context"
	"path/filepath"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestCreateCommand(t *testing.T) {
	sampleFile := filepath.Join("fixture", "rule_type_sample.yaml")
	regoFile := filepath.Join("fixture", "applied_rule.rego")

	tests := []cli.CmdTestCase{
		{
			Name: "create rule type from file",
			Args: []string{"ruletype", "create", "-f", sampleFile},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRuleTypeServiceClient(ctrl)
				mockResp := &minderv1.ListRuleTypesResponse{}
				cli.LoadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					CreateRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateRuleTypeResponse{
						RuleType: mockResp.RuleTypes[0],
					}, nil)
				return cli.WithRPCClient[minderv1.RuleTypeServiceClient](context.Background(), client)
			},
			GoldenFileName: "create_success.table",
		},
		{
			Name: "display server warnings",
			Args: []string{"ruletype", "create", "-f", sampleFile},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRuleTypeServiceClient(ctrl)
				mockResp := &minderv1.ListRuleTypesResponse{}
				cli.LoadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					CreateRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateRuleTypeResponse{
						RuleType: mockResp.RuleTypes[0],
						Warnings: []string{"Rego V0 is deprecated"},
					}, nil)
				return cli.WithRPCClient[minderv1.RuleTypeServiceClient](context.Background(), client)
			},
			GoldenFileName: "create_warning.table",
		},

		{
			Name: "create rule type from rego file",
			Args: []string{"ruletype", "create", "-f", regoFile},
			MockSetup: func(t *testing.T, ctrl *gomock.Controller) context.Context {
				t.Helper()
				client := mockv1.NewMockRuleTypeServiceClient(ctrl)
				mockResp := &minderv1.ListRuleTypesResponse{}
				cli.LoadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					CreateRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateRuleTypeResponse{
						RuleType: mockResp.RuleTypes[0],
					}, nil)
				return cli.WithRPCClient[minderv1.RuleTypeServiceClient](context.Background(), client)
			},
			GoldenFileName: "create_success.table",
		},
		{
			Name:          "missing required file flag",
			Args:          []string{"ruletype", "create"},
			ExpectedError: "required flag(s) \"file\" not set",
		},
	}

	cli.RunCmdTests(t, tests, ruleTypeCmd)
}
