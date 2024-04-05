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

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	_, err := client.AssignRole(ctx, &minderv1.AssignRoleRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
		RoleAssignment: &minderv1.RoleAssignment{
			Role:    r,
			Subject: sub,
		},
	})
	if err != nil {
		return cli.MessageAndError("Error granting role", err)
	}

	cmd.Println("Granted role successfully.")
	return nil
}

func init() {
	RoleCmd.AddCommand(grantCmd)

	grantCmd.Flags().StringP("sub", "s", "", "subject to grant access to")
	grantCmd.Flags().StringP("role", "r", "", "the role to grant")
	if err := grantCmd.MarkFlagRequired("sub"); err != nil {
		grantCmd.Print("Error marking `sub` flag as required.")
		os.Exit(1)
	}
	if err := grantCmd.MarkFlagRequired("role"); err != nil {
		grantCmd.Print("Error marking `role` flag as required.")
		os.Exit(1)
	}
}
