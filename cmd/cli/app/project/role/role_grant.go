// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package role

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var grantCmd = &cobra.Command{
	Use:   "grant",
	Short: "Grant a role to a subject on a project within the minder control plane",
	Long: `The minder project role grant command allows one to grant a role
to a user (subject) on a particular project.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}

		format := viper.GetString("output")

		return app.IsOutputFormatSupported(format)
	},
	RunE: GrantCommand,
}

// GrantCommand is the command for granting roles
func GrantCommand(cmd *cobra.Command, _ []string) error {
	sub := viper.GetString("sub")
	r := viper.GetString("role")
	project := viper.GetString("project")
	email := viper.GetString("email")
	format := viper.GetString("output")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	client, cleanup, err := cli.GetCLIClient(cmd, minderv1.NewPermissionsServiceClient)
	if err != nil {
		return err
	}
	defer cleanup()

	roleAssignment := &minderv1.RoleAssignment{
		Role:    r,
		Subject: sub,
	}
	failMsg := "Error granting role"
	successMsg := "Granted role successfully."
	// Only send an email if one is provided
	if email != "" {
		roleAssignment = &minderv1.RoleAssignment{
			Role:  r,
			Email: email,
		}
		failMsg = "Error creating an invite"
		successMsg = "Invite created successfully."
	}

	resp, err := client.AssignRole(cmd.Context(), &minderv1.AssignRoleRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
		RoleAssignment: roleAssignment,
	})
	if err != nil {
		return cli.MessageAndError(failMsg, err)
	}

	switch format {
	case app.JSON, app.YAML:
		return cli.PrintProto(cmd, format, resp)
	case app.Table:
		fmt.Fprintln(cmd.OutOrStdout(), successMsg)
		if resp.Invitation != nil && resp.Invitation.Code != "" {
			t := initializeTableForGrantListInvitations(cmd.OutOrStdout())
			i := resp.Invitation
			t.AddRow(i.Email, i.Role, i.SponsorDisplay, i.ExpiresAt.AsTime().Format(time.RFC3339))
			t.Render()
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"\nThe invitee can accept it by running: \n\nminder auth invite accept %s\n",
				resp.Invitation.Code,
			)
			if resp.Invitation.InviteUrl != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "\nOr by visiting: %s\n", resp.Invitation.InviteUrl)
			}
		}
	}
	return nil
}

func init() {
	RoleCmd.AddCommand(grantCmd)

	grantCmd.Flags().StringP("sub", "s", "", "subject to grant access to")
	grantCmd.Flags().StringP("role", "r", "", "the role to grant")
	grantCmd.Flags().StringP("email", "e", "", "email to send invitation to")
	grantCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	grantCmd.MarkFlagsOneRequired("sub", "email")
	grantCmd.MarkFlagsMutuallyExclusive("sub", "email")
	if err := grantCmd.MarkFlagRequired("role"); err != nil {
		grantCmd.Print("Error marking `role` flag as required.")
		os.Exit(1)
	}
}
