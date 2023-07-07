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

package repo

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/stacklok/mediator/pkg/auth"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
)

// repo_listCmd represents the list command to list repos with the
// mediator control plane
var repo_listCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories in the mediator control plane",
	Long:  `Repo list is used to register a repo with the mediator control plane`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "error binding flags: %s", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		provider := util.GetConfigValue("provider", "provider", cmd, "").(string)
		if provider != auth.Github {
			return fmt.Errorf("only %s is supported at this time", auth.Github)
		}
		groupID := viper.GetInt32("group-id")
		limit := viper.GetInt32("limit")
		offset := viper.GetInt32("offset")
		format := viper.GetString("output")

		switch format {
		case "json":
		case "yaml":
		case "table":
		case "":
		default:
			return fmt.Errorf("invalid output format: %s", format)
		}

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewRepositoryServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		resp, err := client.ListRepositories(ctx, &pb.ListRepositoriesRequest{
			Provider: provider,
			GroupId:  int32(groupID),
			Limit:    int32(limit),
			Offset:   int32(offset),
			Filter:   pb.RepoFilter_REPO_FILTER_SHOW_REGISTERED_ONLY,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting repo of repos: %s\n", err)
			os.Exit(1)
		}

		switch format {
		case "", "table":
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Id", "Name"})

			for _, v := range resp.Results {
				row := []string{
					fmt.Sprintf("%d", v.GetRepoId()),
					fmt.Sprintf("%s/%s", v.GetOwner(), v.GetName()),
				}
				table.Append(row)
			}
			table.Render()
		case "json":
			output, err := json.MarshalIndent(resp.Results, "", "  ")
			util.ExitNicelyOnError(err, "Error marshalling json")
			fmt.Println(string(output))
		case "yaml":
			yamlData, err := yaml.Marshal(resp.Results)
			util.ExitNicelyOnError(err, "Error marshalling yaml")
			fmt.Println(string(yamlData))
		}
		return nil
	},
}

func init() {
	RepoCmd.AddCommand(repo_listCmd)
	repo_listCmd.Flags().StringP("output", "f", "", "Output format (json or yaml)")
	repo_listCmd.Flags().StringP("provider", "n", "", "Name for the provider to enroll")
	repo_listCmd.Flags().Int32P("group-id", "g", 0, "ID of the group for repo registration")
	repo_listCmd.Flags().Int32P("limit", "l", 20, "Number of repos to display per page")
	repo_listCmd.Flags().Int32P("offset", "o", 0, "Offset of the repos to display")
	if err := repo_listCmd.MarkFlagRequired("provider"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
	if err := repo_listCmd.MarkFlagRequired("group-id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
}
