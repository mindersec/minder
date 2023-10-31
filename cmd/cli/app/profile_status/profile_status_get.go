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

package profile_status

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/cmd/cli/app"
	"github.com/stacklok/mediator/internal/entities"
	"github.com/stacklok/mediator/internal/util"
	minderv1 "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
)

var profilestatus_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get profile status within a mediator control plane",
	Long: `The medic profile_status get subcommand lets you get profile status within a
mediator control plane for an specific provider/project or profile id, entity type and entity id.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := util.GrpcForCommand(cmd)
		if err != nil {
			return fmt.Errorf("error getting grpc connection: %w", err)
		}
		defer conn.Close()

		client := minderv1.NewProfileServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		provider := viper.GetString("provider")
		project := viper.GetString("project")
		profileName := viper.GetString("profile")
		entityId := viper.GetString("entity")
		entityType := viper.GetString("entity-type")
		format := viper.GetString("output")

		switch format {
		case app.JSON, app.YAML, app.Table:
		default:
			return fmt.Errorf("error: invalid format: %s", format)
		}

		if provider == "" {
			return fmt.Errorf("provider must be set")
		}

		req := &minderv1.GetProfileStatusByNameRequest{
			Context: &minderv1.Context{
				Provider: provider,
			},
			Name: profileName,
			Entity: &minderv1.GetProfileStatusByNameRequest_EntityTypedId{
				Id:   entityId,
				Type: minderv1.EntityFromString(entityType),
			},
		}

		if project != "" {
			req.Context.Project = &project
		}

		resp, err := client.GetProfileStatusByName(ctx, req)
		if err != nil {
			return fmt.Errorf("error getting profile status: %w", err)
		}

		switch format {
		case app.JSON:
			out, err := util.GetJsonFromProto(resp)
			util.ExitNicelyOnError(err, "Error getting json from proto")
			fmt.Println(out)
		case app.YAML:
			out, err := util.GetYamlFromProto(resp)
			util.ExitNicelyOnError(err, "Error getting yaml from proto")
			fmt.Println(out)
		case app.Table:
			handleProfileStatusListTable(cmd, resp)
		}

		return nil
	},
}

func init() {
	ProfileStatusCmd.AddCommand(profilestatus_getCmd)
	profilestatus_getCmd.Flags().StringP("provider", "p", "github", "Provider to get profile status for")
	profilestatus_getCmd.Flags().StringP("project", "g", "", "Project ID to get profile status for")
	profilestatus_getCmd.Flags().StringP("profile", "i", "", "Profile name to get profile status for")
	profilestatus_getCmd.Flags().StringP("entity-type", "t", "",
		fmt.Sprintf("the entity type to get profile status for (one of %s)", entities.KnownTypesCSV()))
	profilestatus_getCmd.Flags().StringP("entity", "e", "", "Entity ID to get profile status for")
	profilestatus_getCmd.Flags().StringP("output", "o", app.Table, "Output format (json, yaml or table)")

	// mark as required
	if err := profilestatus_getCmd.MarkFlagRequired("profile"); err != nil {
		util.ExitNicelyOnError(err, "error marking flag as required")
	}
	if err := profilestatus_getCmd.MarkFlagRequired("entity"); err != nil {
		util.ExitNicelyOnError(err, "error marking flag as required")
	}
}
