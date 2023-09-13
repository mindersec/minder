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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/cmd/cli/app"
	github "github.com/stacklok/mediator/internal/providers/github"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

const (
	formatJSON    = app.JSON
	formatYAML    = app.YAML
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

		provider := util.GetConfigValue("provider", "provider", cmd, "").(string)
		repoid := viper.GetInt32("repo-id")
		format := viper.GetString("output")
		name := util.GetConfigValue("name", "name", cmd, "").(string)

		// if name is set, repo-id cannot be set
		if name != "" && repoid != 0 {
			return fmt.Errorf("cannot set both name and repo-id")
		}

		// either name or repoid needs to be set
		if name == "" && repoid == 0 {
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

		conn, err := util.GrpcForCommand(cmd)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewRepositoryServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		// check repo by id
		var repository *pb.RepositoryRecord
		if repoid != 0 {
			resp, err := client.GetRepositoryById(ctx, &pb.GetRepositoryByIdRequest{
				RepositoryId: repoid,
			})
			util.ExitNicelyOnError(err, "Error getting repo by id")
			repository = resp.Repository
		} else {
			if provider != github.Github {
				return fmt.Errorf("only %s is supported at this time", github.Github)
			}

			// check repo by name
			resp, err := client.GetRepositoryByName(ctx, &pb.GetRepositoryByNameRequest{Provider: provider, Name: name})
			util.ExitNicelyOnError(err, "Error getting repo by id")
			repository = resp.Repository
		}

		status := util.GetConfigValue("status", "status", cmd, false).(bool)
		if status {
			// TODO: implement this
		} else {
			// print result just in JSON or YAML
			if format == "" || format == formatJSON {
				out, err := util.GetJsonFromProto(repository)
				util.ExitNicelyOnError(err, "Error getting json from proto")
				fmt.Println(out)
			} else {
				out, err := util.GetYamlFromProto(repository)
				util.ExitNicelyOnError(err, "Error getting json from proto")
				fmt.Println(out)
			}
		}
		return nil
	},
}

func init() {
	RepoCmd.AddCommand(repo_getCmd)
	repo_getCmd.Flags().StringP("output", "f", "", "Output format (json or yaml)")
	repo_getCmd.Flags().StringP("provider", "p", "", "Name for the provider to enroll")
	repo_getCmd.Flags().StringP("name", "n", "", "Name of the repository (owner/name format)")
	repo_getCmd.Flags().Int32P("repo-id", "r", 0, "ID of the repo to query")
	repo_getCmd.Flags().BoolP("status", "s", false, "Only return the status of the policies associated to this repo")
}
