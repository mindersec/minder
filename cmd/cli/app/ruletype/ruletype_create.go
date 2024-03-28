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

	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// createCmd represents the profile create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a rule type",
	Long:  `The ruletype create subcommand lets you create new rule types for a project within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(createCommand),
}

// createCommand is the profile create subcommand
func createCommand(_ context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
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

	createFunc := func(ctx context.Context, _ string, rt *minderv1.RuleType) (*minderv1.RuleType, error) {
		resprt, err := client.CreateRuleType(ctx, &minderv1.CreateRuleTypeRequest{
			RuleType: rt,
		})
		if err != nil {
			return nil, err
		}

		return resprt.RuleType, nil
	}

	for _, f := range files {
		if shouldSkipFile(f) {
			continue
		}
		// cmd.Context() is the root context. We need to create a new context for each file
		// so we can avoid the timeout.
		if err = execOnOneRuleType(cmd.Context(), table, f, os.Stdin, project, createFunc); err != nil {
			return cli.MessageAndError(fmt.Sprintf("Error creating rule type from %s", f), err)
		}
	}
	// Render the table
	table.Render()
	return nil
}
func init() {
	ruleTypeCmd.AddCommand(createCmd)
	// Flags
	createCmd.Flags().StringArrayP("file", "f", []string{},
		"Path to the YAML defining the rule type (or - for stdin). Can be specified multiple times. Can be a directory.")
	// Required
	if err := createCmd.MarkFlagRequired("file"); err != nil {
		createCmd.Printf("Error marking flag required: %s", err)
		os.Exit(1)
	}
}
