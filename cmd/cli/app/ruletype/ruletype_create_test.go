// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global Viper/Stdout state
func TestCreateCommand(t *testing.T) {
	sampleFile := filepath.Join("fixture", "rule_type_sample.yaml")

	tests := []cli.CmdTestCase{
		{
			Name: "create rule type from file",
			Args: []string{"-f", sampleFile},
			MockSetup: func(t *testing.T, client *mockv1.MockRuleTypeServiceClient) {
				t.Helper()
				mockResp := &minderv1.ListRuleTypesResponse{}
				cli.LoadFixture(t, "mock_ruletypes_response.json", mockResp)

				client.EXPECT().
					CreateRuleType(gomock.Any(), gomock.Any()).
					Return(&minderv1.CreateRuleTypeResponse{
						RuleType: mockResp.RuleTypes[0],
					}, nil)
			},
			GoldenFileName: "create_success.table",
		},
		{
			Name: "missing required file flag",
			Args: []string{},
			// Fixed: unused-parameter (changed t to _)
			MockSetup:     func(_ *testing.T, _ *mockv1.MockRuleTypeServiceClient) {},
			ExpectedError: "required flag(s) \"file\" not set",
		},
	}

	execFunc := func(ctx context.Context, cmd *cobra.Command) error {
		if valErr := cmd.ValidateRequiredFlags(); valErr != nil {
			return valErr
		}
		return createCommand(ctx, cmd, cmd.Flags().Args(), nil)
	}

	cli.RunCmdTests(t, tests, createCmd, execFunc)
}
