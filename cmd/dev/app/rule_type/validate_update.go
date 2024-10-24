// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package rule_type

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mindersec/minder/pkg/util/schemaupdate"
)

// CmdValidateUpdate is the command for validating an update of a rule type definition
func CmdValidateUpdate() *cobra.Command {
	var vuCmd = &cobra.Command{
		Use:     "validate-update",
		Aliases: []string{"vu"},
		Short:   "validate an update of a rule type definition",
		Long: `The 'ruletype validate-update' subcommand allows you to validate an update of a rule type 
definition`,
		RunE:         vuCmdRun,
		SilenceUsage: true,
	}
	vuCmd.Flags().StringP("before", "b", "", "file to read rule type definition from")
	vuCmd.Flags().StringP("after", "a", "", "file to read rule type definition from")

	if err := vuCmd.MarkFlagRequired("before"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
	if err := vuCmd.MarkFlagRequired("after"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

	return vuCmd
}

func vuCmdRun(cmd *cobra.Command, _ []string) error {
	beforePath := cmd.Flag("before").Value.String()
	afterPath := cmd.Flag("after").Value.String()

	beforeRt, err := readRuleTypeFromFile(beforePath)
	if err != nil {
		return fmt.Errorf("error reading rule type from %s: %w", beforePath, err)
	}

	afterRt, err := readRuleTypeFromFile(afterPath)
	if err != nil {
		return fmt.Errorf("error reading rule type from %s: %w", afterPath, err)
	}

	// We only validate the after rule type because the before rule type is assumed to be valid
	if err := afterRt.Validate(); err != nil {
		return fmt.Errorf("error validating rule type %s: %w", afterPath, err)
	}

	if beforeRt.GetName() != afterRt.GetName() {
		return fmt.Errorf("rule type name cannot be changed")
	}

	beforeDef := beforeRt.GetDef()
	afterDef := afterRt.GetDef()

	if err := schemaupdate.ValidateSchemaUpdate(beforeDef.GetRuleSchema(), afterDef.GetRuleSchema()); err != nil {
		return fmt.Errorf("error validating rule schema update: %w", err)
	}

	if err := schemaupdate.ValidateSchemaUpdate(beforeDef.GetParamSchema(), afterDef.GetParamSchema()); err != nil {
		return fmt.Errorf("error validating param schema update: %w", err)
	}

	return nil
}
