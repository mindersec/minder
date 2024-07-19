//
// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package role

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var grantCmd = &cobra.Command{
	Use:   "grant",
	Short: "Grant a role to a subject on a project within the minder control plane",
	Long: `The minder project role grant command allows one to grant a role
to a user (subject) on a particular project.`,
	RunE: cli.GRPCClientWrapRunE(GrantCommand),
}

// GrantCommand is the command for granting roles
func GrantCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewPermissionsServiceClient(conn)

	sub := viper.GetString("sub")
	r := viper.GetString("role")
	project := viper.GetString("project")
	email := viper.GetString("email")
	format := viper.GetString("output")

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

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

	resp, err := client.AssignRole(ctx, &minderv1.AssignRoleRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
		RoleAssignment: roleAssignment,
	})
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
		if resp.Invitation != nil && resp.Invitation.Code != "" {
			t := initializeTableForGrantListInvitations()
			i := resp.Invitation
			t.AddRow(i.Email, i.Role, i.SponsorDisplay, i.ExpiresAt.AsTime().Format(time.RFC3339))
			t.Render()
			cmd.Printf("\nThe invitee can accept it by running: \n\nminder auth invite accept %s\n", resp.Invitation.Code)
			if resp.Invitation.InviteUrl != "" {
				cmd.Printf("\nOr by visiting: %s\n", resp.Invitation.InviteUrl)
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
