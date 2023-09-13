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

package artifact

import (
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/auth"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

var artifact_listCmd = &cobra.Command{
	Use:   "list",
	Short: "List artifacts from a provider",
	Long:  `Artifact list will list artifacts from a provider`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "error binding flags: %s", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		format := viper.GetString("output")

		provider := util.GetConfigValue("provider", "provider", cmd, "").(string)
		if provider != auth.Github {
			return fmt.Errorf("only %s is supported at this time", auth.Github)
		}
		groupID := viper.GetInt32("group-id")

		switch format {
		case "json":
		case "yaml":
		case "table":
		case "":
		default:
			return fmt.Errorf("invalid output format: %s", format)
		}

		conn, err := util.GrpcForCommand(cmd)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewArtifactServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		artifacts, err := client.ListArtifacts(
			ctx,
			&pb.ListArtifactsRequest{
				Provider: provider,
				GroupId:  groupID,
			},
		)

		if err != nil {
			return fmt.Errorf("error getting artifacts: %s", err)
		}

		switch format {
		case "", "table":
			table := tablewriter.NewWriter(os.Stdout)

			table.SetHeader([]string{"ID", "Type", "Owner", "Name", "Repository", "Visibility", "Creation date"})

			for _, artifact_item := range artifacts.Results {
				table.Append([]string{
					fmt.Sprintf("%d", artifact_item.ArtifactPk), artifact_item.Type,
					artifact_item.GetOwner(), artifact_item.GetName(),
					artifact_item.Repository,
					artifact_item.Visibility,
					artifact_item.CreatedAt.AsTime().Format(time.RFC3339),
				})
			}

			table.Render()
		case "json":
			out, err := util.GetJsonFromProto(artifacts)
			util.ExitNicelyOnError(err, "Error getting json from proto")
			fmt.Println(out)
		case "yaml":
			out, err := util.GetYamlFromProto(artifacts)
			util.ExitNicelyOnError(err, "Error getting yaml from proto")
			fmt.Println(out)
		}

		return nil
	},
}

func init() {
	ArtifactCmd.AddCommand(artifact_listCmd)
	artifact_listCmd.Flags().StringP("output", "f", "", "Output format (json or yaml)")
	artifact_listCmd.Flags().StringP("provider", "n", "", "Name for the provider to enroll")
	artifact_listCmd.Flags().Int32P("group-id", "g", 0, "ID of the group for repo registration")

	if err := artifact_listCmd.MarkFlagRequired("provider"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
}
