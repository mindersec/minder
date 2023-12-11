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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/cmd/cli/app"
	github "github.com/stacklok/minder/internal/providers/github"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	formatJSON    = app.JSON
	formatYAML    = app.YAML
	formatTable   = "table"
	formatDefault = "" // it actually defaults to table
)

var repo_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get repository in the minder control plane",
	Long:  `Repo get is used to get a repo with the minder control plane`,
	RunE: cli.GRPCClientWrapRunE(func(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
		provider := util.GetConfigValue(viper.GetViper(), "provider", "provider", cmd, "").(string)
		repoid := viper.GetString("repo-id")
		format := viper.GetString("output")
		name := util.GetConfigValue(viper.GetViper(), "name", "name", cmd, "").(string)

		// if name is set, repo-id cannot be set
		if name != "" && repoid != "" {
			return fmt.Errorf("cannot set both name and repo-id")
		}

		// either name or repoid needs to be set
		if name == "" && repoid == "" {
			return fmt.Errorf("either name or repo-id needs to be set")
		}

		// if name is set, provider needs to be set
		if name != "" && provider == "" {
			return fmt.Errorf("provider needs to be set if name is set")
		}

		switch format {
		case formatJSON:
		case formatYAML:
		case formatDefault:
		default:
			return fmt.Errorf("invalid output format: %s", format)
		}

		client := pb.NewRepositoryServiceClient(conn)

		// check repo by id
		var repository *pb.Repository
		if repoid != "" {
			resp, err := client.GetRepositoryById(ctx, &pb.GetRepositoryByIdRequest{
				RepositoryId: repoid,
			})
			if err != nil {
				return cli.MessageAndError(cmd, "Error getting repo by id", err)
			}
			repository = resp.Repository
		} else {
			if provider != github.Github {
				return fmt.Errorf("only %s is supported at this time", github.Github)
			}

			// check repo by name
			resp, err := client.GetRepositoryByName(ctx, &pb.GetRepositoryByNameRequest{Provider: provider, Name: name})
			if err != nil {
				return cli.MessageAndError(cmd, "Error getting repo by name", err)
			}
			repository = resp.Repository
		}

		status := util.GetConfigValue(viper.GetViper(), "status", "status", cmd, false).(bool)
		if status {
			// TODO: implement this
		} else {
			// print result just in JSON or YAML
			if format == "" || format == formatJSON {
				out, err := util.GetJsonFromProto(repository)
				if err != nil {
					return cli.MessageAndError(cmd, "Error getting json from proto", err)
				}
				cli.PrintCmd(cmd, out)
			} else {
				out, err := util.GetYamlFromProto(repository)
				if err != nil {
					return cli.MessageAndError(cmd, "Error getting yaml from proto", err)
				}
				cli.PrintCmd(cmd, out)
			}
		}
		return nil
	}),
}

func init() {
	RepoCmd.AddCommand(repo_getCmd)
	repo_getCmd.Flags().StringP("output", "f", "", "Output format (json or yaml)")
	repo_getCmd.Flags().StringP("provider", "p", "github", "Name for the provider to enroll")
	repo_getCmd.Flags().StringP("name", "n", "", "Name of the repository (owner/name format)")
	repo_getCmd.Flags().StringP("repo-id", "r", "", "ID of the repo to query")
	repo_getCmd.Flags().BoolP("status", "s", false, "Only return the status of the profiles associated to this repo")
}
