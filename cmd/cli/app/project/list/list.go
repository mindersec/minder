//
// Copyright 2024 Stacklok, Inc.
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

// Package list is a subcommand to list projects
package list

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/cmd/cli/app/project"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/cli/table"
	"github.com/stacklok/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// projectListCmd is the root command for the project subcommands
var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	Long:  `List the available projects.`,
	RunE:  cli.GRPCClientWrapRunE(listProjectsCommand),
}

// whoamiCommand is the whoami subcommand
func listProjectsCommand(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	client := minderv1.NewUserServiceClient(conn)

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	userInfo, err := client.GetUser(ctx, &minderv1.GetUserRequest{})
	if err != nil {
		return cli.MessageAndError("Error getting information for user", err)
	}

	renderProjectList(conn.Target(), cmd.OutOrStderr(), viper.GetString("output"), userInfo)
	return nil
}

func renderProjectList(_ string, outWriter io.Writer, format string, user *minderv1.GetUserResponse) {
	switch format {
	case app.Table:
		fmt.Fprintln(outWriter, cli.Header.Render(fmt.Sprintf("Displaying data for %d projects:", len(user.GetProjects()))))
		t := table.New(table.Simple, layouts.KeyValue, nil)

		for _, p := range getProjectTableRows(user.GetProjects()) {
			t.AddRow(p...)
		}
		t.Render()
	case app.JSON:
		out, err := util.GetJsonFromProto(user)
		if err != nil {
			fmt.Fprintf(outWriter, "Error converting to JSON: %v\n", err)
		}
		fmt.Fprintln(outWriter, out)
	case app.YAML:
		out, err := util.GetYamlFromProto(user)
		if err != nil {
			fmt.Fprintf(outWriter, "Error converting to YAML: %v\n", err)
		}
		fmt.Fprintln(outWriter, out)
	}
}

func init() {
	projectListCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	projectListCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
	project.ProjectCmd.AddCommand(projectListCmd)
}

func getProjectTableRows(projects []*minderv1.Project) [][]string {
	var rows [][]string
	for _, p := range projects {
		rows = append(rows, []string{p.GetProjectId(), p.GetName(), p.GetDescription()})
	}
	return rows
}
