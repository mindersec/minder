// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package role

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/mindersec/minder/cmd/cli/app"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/util"
	"github.com/mindersec/minder/pkg/util/cli"
	"github.com/mindersec/minder/pkg/util/cli/table"
	"github.com/mindersec/minder/pkg/util/cli/table/layouts"
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
