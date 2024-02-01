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
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var mapDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a role mapping on a project within the minder control plane",
	Long: `The minder project role mapping delete command allows one to delete role mappings
on a particular project.`,
	RunE: cli.GRPCClientWrapRunE(MapDeleteCommand),
}

// MapDeleteCommand is the command for creating role mappings
func MapDeleteCommand(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	client := minderv1.NewPermissionsServiceClient(conn)

	project := viper.GetString("project")
	id := viper.GetString("id")

	if id == "" {
		return cli.MessageAndError("Role mapping ID is required", errors.New("Role mapping ID is required"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	_, err := client.DeleteRoleMapping(ctx, &minderv1.DeleteRoleMappingRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
	})
	if err != nil {
		return cli.MessageAndError("Error listing roles", err)
	}

	return cli.MessageAndError("Role mapping deleted", nil)
}

func init() {
	mappingCmd.AddCommand(mapDeleteCmd)
	mapDeleteCmd.Flags().StringP("id", "i", "", "Role mapping ID")
}
