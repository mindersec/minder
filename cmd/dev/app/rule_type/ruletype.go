// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package rule_type provides the root command for the ruletype subcommands
package rule_type

import "github.com/spf13/cobra"

// CmdRuleType is the root command for the ruletype subcommands
func CmdRuleType() *cobra.Command {
	var rtCmd = &cobra.Command{
		Use:   "ruletype",
		Short: "ruletype provides utilities for testing rule types",
	}

	rtCmd.AddCommand(CmdTest())
	rtCmd.AddCommand(CmdLint())
	rtCmd.AddCommand(CmdValidateUpdate())
	rtCmd.AddCommand(CmdInit())

	return rtCmd
}
