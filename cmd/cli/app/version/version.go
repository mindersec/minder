// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package version provides the version command for the minder CLI
package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/constants"
	"github.com/mindersec/minder/internal/util/cli/useragent"
)

// VersionCmd is the version command
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print minder CLI version",
	Long:  `The minder version command prints the version of the minder CLI.`,
	Run: func(cmd *cobra.Command, _ []string) {
		fmt.Fprintln(cmd.OutOrStdout(), constants.VerboseCLIVersion)
		fmt.Fprintf(cmd.OutOrStdout(), "User Agent: %s\n", useragent.GetUserAgent())
	},
}

func init() {
	app.RootCmd.AddCommand(VersionCmd)
}
