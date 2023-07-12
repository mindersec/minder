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

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	github "github.com/stacklok/mediator/pkg/providers/github"
)

const (
	formatJSON    = "json"
	formatYAML    = "yaml"
	formatTable   = "table"
	formatDefault = "" // it actually defaults to table
)

// repo_listCmd represents the list command to list repos with the
// mediator control plane
var repo_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get repository in the mediator control plane",
	Long:  `Repo get is used to get a repo with the mediator control plane`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "error binding flags: %s", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		provider := util.GetConfigValue("provider", "provider", cmd, "").(string)
		if provider != github.Github {
			return fmt.Errorf("only %s is supported at this time", github.Github)
		}
		groupID := viper.GetInt32("group-id")
		repoid := viper.GetInt32("repo-id")
		format := viper.GetString("output")

		switch format {
		case formatJSON:
		case formatYAML:
		case formatTable:
		case formatDefault:
		default:
			return fmt.Errorf("invalid output format: %s", format)
		}

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewRepositoryServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		resp, err := client.GetRepository(ctx, &pb.GetRepositoryRequest{
			RepositoryId: repoid,
			Provider:     provider,
			GroupId:      groupID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting repo of repos: %s\n", err)
			os.Exit(1)
		}

		switch format {
		case formatDefault, formatTable:
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Id", "Name", "HookUrl", "Registered", "CreatedAt", "UpdatedAt"})

			row := []string{
				fmt.Sprintf("%d", resp.GetRepoId()),
				fmt.Sprintf("%s/%s", resp.GetOwner(), resp.GetRepository()),
				resp.GetHookUrl(),
				fmt.Sprintf("%t", resp.GetRegistered()),
				resp.GetCreatedAt().AsTime().String(),
				resp.GetUpdatedAt().AsTime().String(),
			}
			table.Append(row)
			table.Render()
		case formatJSON:
			output, err := json.MarshalIndent(resp, "", "  ")
			util.ExitNicelyOnError(err, "Error marshalling json")
			fmt.Println(string(output))
		case formatYAML:
			yamlData, err := yaml.Marshal(resp)
			util.ExitNicelyOnError(err, "Error marshalling yaml")
			fmt.Println(string(yamlData))
		}
		return nil
	},
}

func init() {
	RepoCmd.AddCommand(repo_getCmd)
	repo_getCmd.Flags().StringP("output", "f", "", "Output format (json or yaml)")
	repo_getCmd.Flags().StringP("provider", "n", "", "Name for the provider to enroll")
	repo_getCmd.Flags().Int32P("group-id", "g", 0, "ID of the group for repo registration")
	repo_getCmd.Flags().Int32P("repo-id", "r", 0, "ID of the repo to query")
	if err := repo_getCmd.MarkFlagRequired("provider"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
	if err := repo_getCmd.MarkFlagRequired("group-id"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
}
