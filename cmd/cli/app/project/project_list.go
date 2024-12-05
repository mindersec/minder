// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package project

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

// projectListCmd is the command for listing projects
var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the projects available to you within a minder control plane",
	Long:  `The list command lists the projects available to you within a minder control plane.`,
	RunE:  cli.GRPCClientWrapRunE(listCommand),
}

// listCommand is the command for listing projects
func listCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewProjectsServiceClient(conn)

	format := viper.GetString("output")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	resp, err := client.ListProjects(ctx, &minderv1.ListProjectsRequest{})
	if err != nil {
		return cli.MessageAndError("Error listing projects", err)
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
		t := table.New(table.Simple, layouts.Default, []string{"ID", "Name"})
		for _, v := range resp.Projects {
			t.AddRow(v.ProjectId, v.Name)
		}
		t.Render()
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	return nil
}

func init() {
	ProjectCmd.AddCommand(projectListCmd)
	projectListCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
