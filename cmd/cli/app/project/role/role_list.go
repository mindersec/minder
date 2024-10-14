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

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/internal/util/cli/table"
	"github.com/mindersec/minder/internal/util/cli/table/layouts"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List roles on a project within the minder control plane",
	Long: `The minder project role list command allows one to list roles
available on a particular project.`,
	RunE: cli.GRPCClientWrapRunE(ListCommand),
}

// ListCommand is the command for listing roles
func ListCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
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

	resp, err := client.ListRoles(ctx, &minderv1.ListRolesRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
	})
	if err != nil {
		return cli.MessageAndError("Error listing roles", err)
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
		t := initializeTableForList()
		for _, r := range resp.Roles {
			t.AddRow(r.Name, r.Description)
		}
		t.Render()
	}
	return nil
}

func initializeTableForList() table.Table {
	return table.New(table.Simple, layouts.RoleList, nil)
}

func init() {
	RoleCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
