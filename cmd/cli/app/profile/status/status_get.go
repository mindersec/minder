//
// Copyright 2023 Stacklok, Inc.
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

package status

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/cmd/cli/app/profile"
	"github.com/stacklok/minder/internal/engine/entities"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get profile status",
	Long:  `The profile status get subcommand lets you get profile status within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(getCommand),
}

// getCommand is the profile "get" subcommand
func getCommand(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	client := minderv1.NewProfileServiceClient(conn)

	project := viper.GetString("project")
	profileName := viper.GetString("name")
	entityId := viper.GetString("entity")
	entityType := viper.GetString("entity-type")
	format := viper.GetString("output")

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	resp, err := client.GetProfileStatusByName(ctx, &minderv1.GetProfileStatusByNameRequest{
		Context: &minderv1.Context{Project: &project},
		Name:    profileName,
		Entity: &minderv1.EntityTypedId{
			Id:   entityId,
			Type: minderv1.EntityFromString(entityType),
		},
	})
	if err != nil {
		return cli.MessageAndError("Error getting profile status", err)
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
		table := profile.NewProfileStatusTable()
		profile.RenderProfileStatusTable(resp.ProfileStatus, table)
		table.Render()
	}

	return nil
}

func init() {
	profileStatusCmd.AddCommand(getCmd)
	// Flags
	getCmd.Flags().StringP("entity", "e", "", "Entity ID to get profile status for")
	getCmd.Flags().StringP("entity-type", "t", "",
		fmt.Sprintf("the entity type to get profile status for (one of %s)", entities.KnownTypesCSV()))
	// Required
	if err := getCmd.MarkFlagRequired("entity"); err != nil {
		getCmd.Printf("Error marking flag required: %s", err)
		os.Exit(1)
	}

}
