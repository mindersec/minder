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
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	github "github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var repo_listCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories in the minder control plane",
	Long:  `Repo list is used to register a repo with the minder control plane`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "error binding flags: %s", err)
		}
	},
	RunE: cli.GRPCClientWrapRunE(func(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
		provider := util.GetConfigValue(viper.GetViper(), "provider", "provider", cmd, "").(string)
		if provider != github.Github {
			return fmt.Errorf("only %s is supported at this time", github.Github)
		}
		projectID := viper.GetString("project-id")
		format := viper.GetString("output")

		switch format {
		case "json":
		case "yaml":
		case "table":
		case "":
		default:
			return fmt.Errorf("invalid output format: %s", format)
		}

		client := pb.NewRepositoryServiceClient(conn)

		resp, err := client.ListRepositories(ctx, &pb.ListRepositoriesRequest{
			Provider:  provider,
			ProjectId: projectID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting repo of repos: %s\n", err)
			os.Exit(1)
		}

		switch format {
		case "", "table":
			columns := []table.Column{
				{Title: "ID", Width: 40},
				{Title: "Project", Width: 40},
				{Title: "Provider", Width: 15},
				{Title: "Upstream ID", Width: 15},
				{Title: "Owner", Width: 15},
				{Title: "Name", Width: 15},
			}

			var rows []table.Row
			for _, v := range resp.Results {
				row := table.Row{
					*v.Id,
					*v.Context.Project,
					v.Context.Provider,
					fmt.Sprintf("%d", v.GetRepoId()),
					v.GetOwner(),
					v.GetName(),
				}
				rows = append(rows, row)
			}

			t := table.New(
				table.WithColumns(columns),
				table.WithRows(rows),
				table.WithFocused(false),
				table.WithHeight(len(rows)),
				table.WithStyles(cli.TableHiddenSelectStyles),
			)

			cli.PrintCmd(cmd, cli.TableRender(t))
		case "json":
			out, err := util.GetJsonFromProto(resp)
			util.ExitNicelyOnError(err, "Error getting json from proto")
			fmt.Println(out)
		case "yaml":
			out, err := util.GetYamlFromProto(resp)
			util.ExitNicelyOnError(err, "Error getting yaml from proto")
			fmt.Println(out)
		}
		return nil
	}),
}

func init() {
	RepoCmd.AddCommand(repo_listCmd)
	repo_listCmd.Flags().StringP("output", "f", "", "Output format (json or yaml)")
	repo_listCmd.Flags().StringP("provider", "p", "", "Name for the provider to enroll")
	repo_listCmd.Flags().StringP("project-id", "g", "", "ID of the project for repo registration")
	if err := repo_listCmd.MarkFlagRequired("provider"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
}
