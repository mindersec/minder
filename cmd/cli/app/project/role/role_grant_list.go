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

package role

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

var grantListCmd = &cobra.Command{
	Use:   "list",
	Short: "List role grants within a given project",
	Long: `The minder project role grant list command lists all role grants
on a particular project.`,
	RunE: cli.GRPCClientWrapRunE(GrantListCommand),
}

// GrantListCommand is the command for listing grants
func GrantListCommand(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	client := minderv1.NewPermissionsServiceClient(conn)

	project := viper.GetString("project")
	format := viper.GetString("output")
	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	resp, err := client.ListRoleAssignments(ctx, &minderv1.ListRoleAssignmentsRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
	})
	if err != nil {
		return cli.MessageAndError("Error listing role grants", err)
	}

	switch format {
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
	case app.Table:
		t := initializeTableForGrantList()
		for _, r := range resp.RoleAssignments {
			t.AddRow(r.Subject, r.Role, r.GetMapping().GetId(),
				structtoYAMLOrEmpty(r.GetMapping().GetClaimsToMatch()),
			)
		}
		for _, r := range resp.UnmatchedMappings {
			t.AddRow("", r.Role, r.GetId(),
				structtoYAMLOrEmpty(r.GetClaimsToMatch()),
			)
		}
		t.Render()
	}
	return nil
}

func initializeTableForGrantList() table.Table {
	return table.New(table.Simple, layouts.Default, []string{"Subject", "Role", "Mapping ID", "Mapping"})
}

func init() {
	grantCmd.AddCommand(grantListCmd)
	grantListCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
