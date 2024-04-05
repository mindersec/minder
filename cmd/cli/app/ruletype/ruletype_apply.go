//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ruletype

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a rule type",
	Long:  `The ruletype apply subcommand lets you create or update rule types for a project within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(applyCommand),
}

// applyCommand is the "rule type" apply subcommand
func applyCommand(_ context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewProfileServiceClient(conn)

	project := viper.GetString("project")

	fileFlag, err := cmd.Flags().GetStringArray("file")
	if err != nil {
		return cli.MessageAndError("Error parsing file flag", err)
	}

	if err = validateFilesArg(fileFlag); err != nil {
		return cli.MessageAndError("Error validating file flag", err)
	}

	files, err := util.ExpandFileArgs(fileFlag)
	if err != nil {
		return cli.MessageAndError("Error expanding file args", err)
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	table := initializeTableForList()

	applyFunc := func(ctx context.Context, fileName string, rt *minderv1.RuleType) (*minderv1.RuleType, error) {
		createResp, err := client.CreateRuleType(ctx, &minderv1.CreateRuleTypeRequest{
			RuleType: rt,
		})

		if err == nil {
			return createResp.RuleType, nil
		}

		st, ok := status.FromError(err)
		if !ok {
			// We can't parse the error, so just return it
			return nil, fmt.Errorf("error creating rule type from %s: %w", fileName, err)
		}

		if st.Code() != codes.AlreadyExists {
			return nil, fmt.Errorf("error creating rule type from %s: %w", fileName, err)
		}

		updateResp, err := client.UpdateRuleType(ctx, &minderv1.UpdateRuleTypeRequest{
			RuleType: rt,
		})

		if err != nil {
			return nil, fmt.Errorf("error updating rule type from %s: %w", fileName, err)
		}

		return updateResp.RuleType, nil
	}

	for _, f := range files {
		if shouldSkipFile(f) {
			continue
		}
		// cmd.Context() is the root context. We need to create a new context for each file
		// so we can avoid the timeout.
		if err = execOnOneRuleType(cmd.Context(), table, f, os.Stdin, project, applyFunc); err != nil {
			return cli.MessageAndError(fmt.Sprintf("error applying rule type from %s", f), err)
		}
	}
	// Render the table
	table.Render()
	return nil
}

func init() {
	ruleTypeCmd.AddCommand(applyCmd)
	// Flags
	applyCmd.Flags().StringArrayP("file", "f", []string{},
		"Path to the YAML defining the rule type (or - for stdin). Can be specified multiple times. Can be a directory.")
	// Required
	if err := applyCmd.MarkFlagRequired("file"); err != nil {
		applyCmd.Printf("Error marking flag required: %s", err)
		os.Exit(1)
	}
}
