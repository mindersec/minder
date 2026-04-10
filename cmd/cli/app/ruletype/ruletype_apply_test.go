// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestApplyCommand(t *testing.T) {
	applyFixture := filepath.Join("fixture", "rule_type_apply.yaml")

	tests := []cli.CmdTestCase{
		{
			Name: "apply - create new rule type via flag",
			Args: []string{"-f", applyFixture},
			MockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				t.Helper()
				mockResp := &minderv1.ListRuleTypesResponse{}
				cli.LoadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					CreateRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateRuleTypeResponse{RuleType: mockResp.RuleTypes[0]}, nil)
			},
			GoldenFileName: "apply_create.table",
		},
		{
			Name: "apply - update existing rule type via positional arg",
			Args: []string{applyFixture},
			MockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				t.Helper()
				mockResp := &minderv1.ListRuleTypesResponse{}
				cli.LoadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					CreateRuleType(gomock.Any(), gomock.Any()).
					Return(nil, status.Error(codes.AlreadyExists, "already exists"))

				client.EXPECT().
					UpdateRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.UpdateRuleTypeResponse{RuleType: mockResp.RuleTypes[0]}, nil)
			},
			GoldenFileName: "apply_update.table",
		},
		{
			Name:          "no files specified",
			Args:          []string{},
			MockSetup:     func(_ *testing.T, _ *mockv1.MockRuleTypeServiceClient) {},
			ExpectedError: "no files specified",
		},
	}

	execFunc := func(ctx context.Context, cmd *cobra.Command) error {
		return applyCommand(ctx, cmd, cmd.Flags().Args(), nil)
	}

	cli.RunCmdTests(t, tests, applyCmd, execFunc)
}
