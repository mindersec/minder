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
	"context"
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var artifact_listCmd = &cobra.Command{
	Use:   "list",
	Short: "List artifacts from a provider",
	Long:  `Artifact list will list artifacts from a provider`,
	RunE: cli.GRPCClientWrapRunE(func(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
		format := viper.GetString("output")

		provider := util.GetConfigValue(viper.GetViper(), "provider", "provider", cmd, "").(string)
		projectID := viper.GetString("project-id")

		switch format {
		case "json":
		case "yaml":
		case "table":
		case "":
		default:
			return fmt.Errorf("invalid output format: %s", format)
		}

		client := pb.NewArtifactServiceClient(conn)

		artifacts, err := client.ListArtifacts(
			ctx,
			&pb.ListArtifactsRequest{
				Provider:  provider,
				ProjectId: projectID,
				From:      viper.GetString("from"),
			},
		)

		if err != nil {
			return cli.MessageAndError(cmd, "Couldn't list artifacts", err)
		}

		switch format {
		case "", "table":
			table := tablewriter.NewWriter(os.Stdout)

			table.SetHeader([]string{"ID", "Type", "Owner", "Name", "Repository", "Visibility", "Creation date"})

			for _, artifact_item := range artifacts.Results {
				table.Append([]string{
					artifact_item.ArtifactPk,
					artifact_item.Type,
					artifact_item.GetOwner(),
					artifact_item.GetName(),
					artifact_item.Repository,
					artifact_item.Visibility,
					artifact_item.CreatedAt.AsTime().Format(time.RFC3339),
				})
			}

			table.Render()
		case "json":
			out, err := util.GetJsonFromProto(artifacts)
			if err != nil {
				return cli.MessageAndError(cmd, "Error getting json from proto", err)
			}
			cmd.Println(out)
		case "yaml":
			out, err := util.GetYamlFromProto(artifacts)
			if err != nil {
				return cli.MessageAndError(cmd, "Error getting yaml from proto", err)
			}
			cmd.Println(out)
		}

		return nil
	}),
}

func init() {
	ArtifactCmd.AddCommand(artifact_listCmd)
	artifact_listCmd.Flags().StringP("output", "f", "", "Output format (json or yaml)")
	artifact_listCmd.Flags().StringP("provider", "p", "github", "Name for the provider to enroll")
	artifact_listCmd.Flags().StringP("project-id", "g", "", "ID of the project for repo registration")
	artifact_listCmd.Flags().String("from", "", "Filter artifacts from a source, example: from=repository=owner/repo")
}
