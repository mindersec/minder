// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var applyCmd = &cobra.Command{
	Use:   "apply [files...]",
	Short: "Apply a rule type",
	Long:  `The ruletype apply subcommand lets you create or update rule types for a project within Minder.`,
	Args: func(cmd *cobra.Command, args []string) error {
		fileFlag, err := cmd.Flags().GetStringArray("file")
		if err != nil {
			return cli.MessageAndError("Error parsing file flag", err)
		}

		if len(fileFlag) == 0 && len(args) == 0 {
			return fmt.Errorf("no files specified: use positional arguments or the -f flag")
		}
		return nil
	},
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %s", err)
		}

		fileFlag, _ := cmd.Flags().GetStringArray("file")

		if err := validateFilesArg(fileFlag); err != nil {
			return cli.MessageAndError("Error validating files", err)
		}

		return nil
	},
	RunE: applyCommand,
}

func applyCommand(cmd *cobra.Command, args []string) error {
	fileFlag, _ := cmd.Flags().GetStringArray("file")

	// Combine positional args with -f flag values
	allFiles := append(fileFlag, args...)

	files, err := util.ExpandFileArgs(allFiles...)
	if err != nil {
		return cli.MessageAndError("Error expanding file args", err)
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	client, closeConn, err := getRuleTypeClient(cmd)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer closeConn()

	project := viper.GetString("project")

	table := initializeTableForList(cmd.OutOrStdout())

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
		if f.Path != "-" && shouldSkipFile(f.Path) {
			continue
		}
		if err = execOnOneRuleType(cmd.Context(), table, f.Path, os.Stdin, project, applyFunc); err != nil {
			if f.Expanded && minderv1.YouMayHaveTheWrongResource(err) {
				cmd.PrintErrf("Skipping file %s: not a rule type\n", f.Path)
				// We'll skip the file if it's not a rule type
				continue
			}
			return cli.MessageAndError(fmt.Sprintf("error applying rule type from %s", f.Path), err)
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
}
