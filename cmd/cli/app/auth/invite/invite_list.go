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

// inviteListCmd represents the list command
var inviteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pending invitations",
	Long:  `List shows all pending invitations for the current minder user`,
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
	RunE: inviteListCommand,
}

// inviteListCommand is the invite list subcommand
func inviteListCommand(cmd *cobra.Command, _ []string) error {
	client, cleanup, err := cli.GetCLIClient(cmd, minderv1.NewUserServiceClient)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer cleanup()

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true
	format := viper.GetString("output")

	res, err := client.ListInvitations(cmd.Context(), &minderv1.ListInvitationsRequest{})
	if err != nil {
		return cli.MessageAndError("Error listing invitations", err)
	}

	switch format {
	case app.JSON, app.YAML:
		return cli.PrintProto(cmd, format, res)
	case app.Table:
		if len(res.Invitations) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No pending invitations")
			return nil
		}
		t := table.New(
			table.Simple, layouts.Default, cmd.OutOrStdout(),
			[]string{"Sponsor", "Project", "Role", "Expires", "Code"},
		)
		for _, v := range res.Invitations {
			t.AddRow(v.SponsorDisplay, v.Project, v.Role, v.ExpiresAt.AsTime().Format(time.RFC3339), v.Code)
		}
		t.Render()
	}
	return nil
}

func init() {
	inviteCmd.AddCommand(inviteListCmd)
	inviteListCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
