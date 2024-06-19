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

	sub := viper.GetString("sub")
	r := viper.GetString("role")
	project := viper.GetString("project")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	ret, err := client.UpdateRole(ctx, &minderv1.UpdateRoleRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
		Role:    []string{r},
		Subject: sub,
	})
	if err != nil {
		return cli.MessageAndError("Error updating role", err)
	}

	cmd.Println("Update role successfully.")
	cmd.Printf(
		"Subject \"%s\" is now assigned to role \"%s\" on project \"%s\"\n",
		ret.RoleAssignments[0].Subject,
		ret.RoleAssignments[0].Role,
		*ret.RoleAssignments[0].Project,
	)

	return nil
}

func init() {
	RoleCmd.AddCommand(updateCmd)

	updateCmd.Flags().StringP("sub", "s", "", "subject to update role access for")
	updateCmd.Flags().StringP("role", "r", "", "the role to update it to")
	updateCmd.MarkFlagsRequiredTogether("sub", "role")
}
