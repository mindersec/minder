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
	"github.com/stacklok/minder/internal/util/cli/table"
	"github.com/stacklok/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories",
	Long:  `The repo list subcommand is used to list registered repositories within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(listCommand),
}

// listCommand is the repo list subcommand
func listCommand(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	client := minderv1.NewRepositoryServiceClient(conn)

	provider := viper.GetString("provider")
	project := viper.GetString("project")
	format := viper.GetString("output")

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	resp, err := client.ListRepositories(ctx, &minderv1.ListRepositoriesRequest{
		Context: &minderv1.Context{Provider: &provider, Project: &project},
	})
	if err != nil {
		return cli.MessageAndError("Error listing repositories", err)
	}

	switch format {
	case app.Table:
		t := table.New(table.Simple, layouts.RepoList, nil)
		for _, v := range resp.Results {
			t.AddRow(
				*v.Id,
				*v.Context.Project,
				*v.Context.Provider,
				fmt.Sprintf("%d", v.GetRepoId()),
				v.GetOwner(),
				v.GetName(),
			)
		}
		t.Render()
	case app.JSON:
		out, err := util.GetJsonFromProto(resp)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(resp)
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	}
	return nil
}

func init() {
	RepoCmd.AddCommand(listCmd)
	// Flags
	listCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
