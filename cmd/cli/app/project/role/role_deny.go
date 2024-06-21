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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var denyCmd = &cobra.Command{
	Use:   "deny",
	Short: "Deny a role to a subject on a project within the minder control plane",
	Long: `The minder project role deny command removes a user from a role grant
on a particular project.`,
	RunE: cli.GRPCClientWrapRunE(DenyCommand),
}

// DenyCommand is the command for removing a role assignment from a project
func DenyCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewPermissionsServiceClient(conn)

	sub := viper.GetString("sub")
	r := viper.GetString("role")
	project := viper.GetString("project")
	email := viper.GetString("email")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	roleAssignment := &minderv1.RoleAssignment{
		Role:    r,
		Subject: sub,
	}
	failMsg := "Error denying role"
	successMsg := "Denied role successfully."
	// Only send an email if one is provided
	if email != "" {
		roleAssignment = &minderv1.RoleAssignment{
			Role:  r,
			Email: email,
		}
		failMsg = "Error deleting an invite"
		successMsg = "Invite deleted successfully."
	}

	_, err := client.RemoveRole(ctx, &minderv1.RemoveRoleRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
		RoleAssignment: roleAssignment,
	})
	if err != nil {
		return cli.MessageAndError(failMsg, err)
	}

	cmd.Println(successMsg)
	return nil
}

func init() {
	RoleCmd.AddCommand(denyCmd)

	denyCmd.Flags().StringP("role", "r", "", "the role to grant")
	denyCmd.Flags().StringP("sub", "s", "", "subject to grant access to")
	denyCmd.Flags().StringP("email", "e", "", "email to send invitation to")
	denyCmd.MarkFlagsOneRequired("sub", "email")
	denyCmd.MarkFlagsMutuallyExclusive("sub", "email")
	if err := denyCmd.MarkFlagRequired("role"); err != nil {
		denyCmd.Print("Error marking `role` flag as required.")
		os.Exit(1)
	}
}
