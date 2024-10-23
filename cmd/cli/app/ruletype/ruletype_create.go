// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// createCmd represents the profile create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a rule type",
	Long:  `The ruletype create subcommand lets you create new rule types for a project within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(createCommand),
}

// createCommand is the profile create subcommand
func createCommand(_ context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewRuleTypeServiceClient(conn)

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
		if shouldSkipFile(f.Path) {
			continue
		}
		// cmd.Context() is the root context. We need to create a new context for each file
		// so we can avoid the timeout.
		if err = execOnOneRuleType(cmd.Context(), table, f.Path, os.Stdin, project, createFunc); err != nil {
			// We swallow errors if you're loading a directory to avoid failing
			// on test files.
			if f.Expanded && minderv1.YouMayHaveTheWrongResource(err) {
				cmd.PrintErrf("Skipping file %s: not a rule type\n", f.Path)
				// We'll skip the file if it's not a rule type
				continue
			}
			return cli.MessageAndError(fmt.Sprintf("Error creating rule type from %s", f.Path), err)
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
