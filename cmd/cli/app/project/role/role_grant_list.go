// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package role

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/internal/util/cli/table"
	"github.com/mindersec/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var grantListCmd = &cobra.Command{
	Use:   "list",
	Short: "List role grants within a given project",
	Long: `The minder project role grant list command lists all role grants
on a particular project.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return cli.MessageAndError("Error binding flags", err)
		}
		return nil
	},
	RunE: GrantListCommand,
}

// GrantListCommand is the command for listing grants
func GrantListCommand(cmd *cobra.Command, _ []string) error {
	client, cleanup, err := GetPermissionsClient(cmd)
	if err != nil {
		return cli.MessageAndError("Error getting client", err)
	}
	defer cleanup()

	project := viper.GetString("project")
	format := viper.GetString("output")

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	resp, err := client.ListRoleAssignments(cmd.Context(), &minderv1.ListRoleAssignmentsRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
	})
	if err != nil {
		return cli.MessageAndError("Error listing role grants", err)
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
		t := initializeTableForGrantListRoleAssignments(cmd.OutOrStdout())
		for _, r := range resp.RoleAssignments {
			t.AddRow(fmt.Sprintf("%s / %s", r.DisplayName, r.Subject), r.Role, *r.Project)
		}
		t.Render()
		if len(resp.Invitations) > 0 {
			t2 := initializeTableForGrantListInvitations(cmd.OutOrStdout())
			for _, r := range resp.Invitations {
				t2.AddRow(r.Email, r.Role, r.SponsorDisplay, r.ExpiresAt.AsTime().Format(time.RFC3339))
			}
			t2.Render()
		} else {
			cmd.Println("No pending invitations found.")
		}
	}
	return nil
}

func initializeTableForGrantListRoleAssignments(out io.Writer) table.Table {
	return table.New(table.Simple, layouts.Default, out, []string{"User", "Role", "Project"})
}

func initializeTableForGrantListInvitations(out io.Writer) table.Table {
	return table.New(table.Simple, layouts.Default, out, []string{"Invitee", "Role", "Sponsor", "Expires At"})
}

func init() {
	grantCmd.AddCommand(grantListCmd)
	grantListCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
