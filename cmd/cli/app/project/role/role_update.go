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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "update a role to a subject on a project",
	Long: `The minder project role update command allows one to update a role
to a user (subject) on a particular project.`,
	RunE: cli.GRPCClientWrapRunE(UpdateCommand),
}

// UpdateCommand is the command for granting roles
func UpdateCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewPermissionsServiceClient(conn)

	r := viper.GetString("role")
	project := viper.GetString("project")
	sub := viper.GetString("sub")
	email := viper.GetString("email")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	req := &minderv1.UpdateRoleRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
		Roles:   []string{r},
		Subject: sub,
	}
	failMsg := "Error updating role"
	successMsg := "Updated role successfully."
	if email != "" {
		req.Email = email
		failMsg = "Error updating an invite"
		successMsg = "Invite updated successfully."
	}

	ret, err := client.UpdateRole(ctx, req)
	if err != nil {
		return cli.MessageAndError(failMsg, err)
	}

	cmd.Println(successMsg)

	// If it was an invitation, print the invite details
	if len(ret.Invitations) != 0 {
		for _, r := range ret.Invitations {
			// TODO: Add a url to the invite
			cmd.Printf("Updated an invite for %s to %s on %s\n\nThe invitee can accept it by running: \n\nminder auth invite accept %s\n",
				r.Email, r.Role, r.Project, r.Code)
		}
		return nil
	}
	// Otherwise, print the role assignments if it was about updating a role
	t := initializeTableForGrantListRoleAssignments()
	for _, r := range ret.RoleAssignments {
		t.AddRow(fmt.Sprintf("%s / %s", r.DisplayName, r.Subject), r.Role, *r.Project)
	}
	t.Render()
	return nil
}

func init() {
	RoleCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringP("role", "r", "", "the role to update it to")
	updateCmd.Flags().StringP("sub", "s", "", "subject to update role access for")
	updateCmd.Flags().StringP("email", "e", "", "email to send invitation to")
	updateCmd.MarkFlagsOneRequired("sub", "email")
	updateCmd.MarkFlagsMutuallyExclusive("sub", "email")
}
