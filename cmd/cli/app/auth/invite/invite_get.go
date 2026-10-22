// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package invite provides the auth invite command for the minder CLI.
package invite

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/internal/util/cli/table"
	"github.com/mindersec/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// inviteGetCmd represents the list command
var inviteGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get info about pending invitations",
	Long:  `Get shows additional information about a pending invitation`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}

		format := viper.GetString("output")

		// Ensure the output format is supported
		if !app.IsOutputFormatSupported(format) {
			return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
		}

		return nil
	},
	RunE: inviteGetCommand,
	Args: cobra.ExactArgs(1),
}

// inviteGetCommand is the invite get subcommand
func inviteGetCommand(cmd *cobra.Command, args []string) error {
	client, cleanup, err := cli.GetCLIClient(cmd, minderv1.NewInviteServiceClient)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer cleanup()

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true
	format := viper.GetString("output")

	res, err := client.GetInviteDetails(cmd.Context(), &minderv1.GetInviteDetailsRequest{
		Code: args[0],
	})
	if err != nil {
		return cli.MessageAndError("Error getting info for invitation", err)
	}

	switch format {
	case app.JSON, app.YAML:
		return cli.PrintProto(cmd, format, res)
	case app.Table:
		t := table.New(table.Simple, layouts.Default, cmd.OutOrStdout(), []string{"Sponsor", "Project", "Expires"})
		t.AddRow(res.SponsorDisplay, res.ProjectDisplay, res.ExpiresAt.AsTime().Format(time.RFC3339))
		t.Render()
	}
	return nil
}

func init() {
	inviteCmd.AddCommand(inviteGetCmd)
	inviteGetCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
