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
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get repository details",
	Long:  `The repo get subcommand is used to get details for a registered repository within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(getCommand),
}

// getCommand is the repo get subcommand
func getCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewRepositoryServiceClient(conn)

	provider := viper.GetString("provider")
	project := viper.GetString("project")
	format := viper.GetString("output")
	repoid := viper.GetString("id")
	name := viper.GetString("name")

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) || format == app.Table {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	var repository *minderv1.Repository
	// check repo by id
	if repoid != "" {
		resp, err := client.GetRepositoryById(ctx, &minderv1.GetRepositoryByIdRequest{
			Context:      &minderv1.Context{Provider: &provider, Project: &project},
			RepositoryId: repoid,
		})
		if err != nil {
			return cli.MessageAndError("Error getting repo by id", err)
		}
		repository = resp.Repository
	} else {
		// check repo by name
		resp, err := client.GetRepositoryByName(ctx, &minderv1.GetRepositoryByNameRequest{
			Context: &minderv1.Context{Provider: &provider, Project: &project},
			Name:    name,
		})
		if err != nil {
			return cli.MessageAndError("Error getting repo by name", err)
		}
		repository = resp.Repository
	}

	// print result just in JSON or YAML
	switch format {
	case app.JSON:
		out, err := util.GetJsonFromProto(repository)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(repository)
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	}

	return nil
}
func init() {
	RepoCmd.AddCommand(getCmd)
	// Flags
	getCmd.Flags().StringP("output", "o", app.JSON,
		fmt.Sprintf("Output format (one of %s)", strings.Join([]string{app.JSON, app.YAML}, ",")))
	getCmd.Flags().StringP("name", "n", "", "Name of the repository (owner/name format)")
	getCmd.Flags().StringP("id", "i", "", "ID of the repo to query")
	// Required
	getCmd.MarkFlagsOneRequired("name", "id")
	getCmd.MarkFlagsMutuallyExclusive("name", "id")
}
