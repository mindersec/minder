// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package app provides the root command for the mindev CLI
package app

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/mindersec/minder/cmd/dev/app/bundles"
	"github.com/mindersec/minder/cmd/dev/app/datasource"
	"github.com/mindersec/minder/cmd/dev/app/image"
	"github.com/mindersec/minder/cmd/dev/app/rule_type"
	"github.com/mindersec/minder/cmd/dev/app/testserver"
	"github.com/mindersec/minder/internal/util/cli"
)

// CmdRoot represents the base command when called without any subcommands
func CmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mindev",
		Short: "mindev provides developer tooling for minder",
		Long: `For more information about minder, please visit:
https://docs.stacklok.com/minder`,
	}

	cmd.AddCommand(rule_type.CmdRuleType())
	cmd.AddCommand(image.CmdImage())
	cmd.AddCommand(testserver.CmdTestServer())
	cmd.AddCommand(bundles.CmdBundle())
	cmd.AddCommand(datasource.CmdDataSource())

	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	cmd := CmdRoot()
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	err := cmd.Execute()
	cli.ExitNicelyOnError(err, "Error on execute")
}
