// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"context"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mindersec/minder/internal/util/cli"
	mockv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1/mock"
)

//nolint:paralleltest // Cannot run in parallel because it swaps global state
func TestRuleTypeRootCommand(t *testing.T) {
	tests := []cli.CmdTestCase{
		{
			Name:           "ruletype root command shows help",
			Args:           []string{},
			MockSetup:      func(_ *testing.T, _ *mockv1.MockRuleTypeServiceClient) {},
			GoldenFileName: "ruletype_root.help",
		},
	}

	execFunc := func(_ context.Context, cmd *cobra.Command) error {
		return cmd.RunE(cmd, []string{})
	}

	cli.RunCmdTests(t, tests, ruleTypeCmd, execFunc)
}
