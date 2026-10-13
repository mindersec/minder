// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package project

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}

		format := viper.GetString("output")

		// Ensure the output format is supported
		if !app.IsOutputFormatSupported(format) {
			return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
		}

		return nil
	},
	RunE: listCommand,
}

// listCommand is the command for listing projects
func listCommand(cmd *cobra.Command, _ []string) error {
	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	client, cleanup, err := GetProjectsClient(cmd)
	if err != nil {
		return err
	}
	defer cleanup()

	format := viper.GetString("output")

	resp, err := client.ListProjects(cmd.Context(), &minderv1.ListProjectsRequest{})
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
		t := table.New(table.Simple, layouts.Default, cmd.OutOrStdout(), []string{"ID", "Name"}).SetAutoMerge(true)

		for _, v := range resp.Projects {
			t.AddRow(v.ProjectId, v.Name)
		}
		t.Render()
	}

	return nil
}

func init() {
	ProjectCmd.AddCommand(projectListCmd)
	projectListCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
