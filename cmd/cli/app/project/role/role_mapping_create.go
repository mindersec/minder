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
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var mapCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a role mapping on a project within the minder control plane",
	Long: `The minder project role mapping create command allows one to create role mappings
on a particular project.`,
	RunE: cli.GRPCClientWrapRunE(MapCreateCommand),
}

// MapCreateCommand is the command for creating role mappings
func MapCreateCommand(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	client := minderv1.NewPermissionsServiceClient(conn)

	project := viper.GetString("project")
	role := viper.GetString("role")
	mappings := viper.GetString("mappings")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	m := &structpb.Struct{}

	if err := m.UnmarshalJSON([]byte(mappings)); err != nil {
		return cli.MessageAndError("Error unmarshalling mappings", err)
	}

	_, err := client.CreateRoleMapping(ctx, &minderv1.CreateRoleMappingRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
		RoleMapping: &minderv1.RoleMapping{
			Role:          role,
			ClaimsToMatch: m,
		},
	})
	if err != nil {
		return cli.MessageAndError("Error listing roles", err)
	}

	return cli.MessageAndError("Role mapping created", nil)
}

func init() {
	mappingCmd.AddCommand(mapCreateCmd)
	mapCreateCmd.Flags().StringP("role", "r", "", "Role to map")
	mapCreateCmd.Flags().StringP("mappings", "m", "", "JSON string of claim mappings to create")

	// TODO: Add support for reading role mapping from file
	// mapCreateCmd.Flags().StringP("file", "f", "", "File containing role mapping to create")
}
