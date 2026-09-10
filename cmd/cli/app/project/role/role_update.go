// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package role

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "update a role to a subject on a project",
	Long: `The minder project role update command allows one to update a role
to a user (subject) on a particular project.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return cli.MessageAndError("Error binding flags", err)
		}
		return nil
	},
	RunE: UpdateCommand,
}

// UpdateCommand is the command for granting roles
func UpdateCommand(cmd *cobra.Command, _ []string) error {
	client, cleanup, err := GetPermissionsClient(cmd)
	if err != nil {
		return cli.MessageAndError("Error getting client", err)
	}
	defer cleanup()

	role := viper.GetString("role")
	project := viper.GetString("project")
	sub := viper.GetString("sub")
	email := viper.GetString("email")
	format := viper.GetString("output")

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	req := &minderv1.UpdateRoleRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
		Roles:   []string{role},
		Subject: sub,
	}
	failMsg := "Error updating role"
	successMsg := "Updated role successfully."
	if email != "" {
		req.Email = email
		failMsg = "Error updating an invite"
		successMsg = "Invite updated successfully."
	}

	resp, err := client.UpdateRole(cmd.Context(), req)
	if err != nil {
		return cli.MessageAndError(failMsg, err)
	}

	switch format {
	case app.JSON:
		out, err := util.GetJsonFromProto(resp)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(resp)
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	case app.Table:
		cmd.Println(successMsg)
		// If it was an invitation, print the invite details
		if len(resp.Invitations) != 0 {
			for _, r := range resp.Invitations {
				cmd.Printf("Updated an invite for %s to %s on %s\n\nThe invitee can accept it by running: \n\nminder auth invite accept %s\n",
					r.Email, r.Role, r.Project, r.Code)
				if r.InviteUrl != "" {
					cmd.Printf("\nOr by visiting: %s\n", r.InviteUrl)
				}
			}
			return nil
		}
		// Otherwise, print the role assignments if it was about updating a role
		t := initializeTableForGrantListRoleAssignments(cmd.OutOrStdout())
		for _, r := range resp.RoleAssignments {
			t.AddRow(fmt.Sprintf("%s / %s", r.DisplayName, r.Subject), r.Role, *r.Project)
		}
		t.Render()
	}
	return nil
}

func init() {
	RoleCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringP("role", "r", "", "the role to update it to")
	updateCmd.Flags().StringP("sub", "s", "", "subject to update role access for")
	updateCmd.Flags().StringP("email", "e", "", "email to send invitation to")
	updateCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	updateCmd.MarkFlagsOneRequired("sub", "email")
	updateCmd.MarkFlagsMutuallyExclusive("sub", "email")
}
