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

package profile

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/cmd/cli/app"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
)

var profile_listCmd = &cobra.Command{
	Use:   "list",
	Short: "List profiles within a minder control plane",
	Long: `The minder profile list subcommand lets you list profiles within a
minder control plane for an specific project.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		format := viper.GetString("output")

		conn, err := util.GrpcForCommand(cmd)
		if err != nil {
			return fmt.Errorf("error getting grpc connection: %w", err)
		}
		defer conn.Close()

		client := pb.NewProfileServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		provider := viper.GetString("provider")

		switch format {
		case app.JSON:
		case app.YAML:
		case app.Table:
		default:
			return fmt.Errorf("invalid format: %s", format)
		}

		resp, err := client.ListProfiles(ctx, &pb.ListProfilesRequest{
			Context: &pb.Context{
				Provider: provider,
				// TODO set up project if specified
				// Currently it's inferred from the authorization token
			},
		})
		if err != nil {
			return fmt.Errorf("error getting profiles: %w", err)
		}

		switch format {
		case app.JSON:
			out, err := util.GetJsonFromProto(resp)
			util.ExitNicelyOnError(err, "Error getting json from proto")
			fmt.Println(out)
		case app.YAML:
			out, err := util.GetYamlFromProto(resp)
			util.ExitNicelyOnError(err, "Error getting json from proto")
			fmt.Println(out)
		case app.Table:
			handleListTableOutput(cmd, resp)
			return nil
		}

		// this is unreachable
		return nil
	},
}

func init() {
	ProfileCmd.AddCommand(profile_listCmd)
	profile_listCmd.Flags().StringP("provider", "p", "", "Provider to list profiles for")
	profile_listCmd.Flags().StringP("output", "o", app.Table, "Output format (json, yaml or table)")
	// TODO: Take project ID into account
	// profile_listCmd.Flags().Int32P("project-id", "g", 0, "project id to list roles for")

	if err := profile_listCmd.MarkFlagRequired("provider"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
}

func handleListTableOutput(cmd *cobra.Command, resp *pb.ListProfilesResponse) {
	table := initializeTable(cmd)

	for _, v := range resp.Profiles {
		renderProfileTable(v, table)
	}
	table.Render()
}
