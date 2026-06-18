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

// projectCreateCmd is the command for creating sub-projects
var projectCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a sub-project within a minder control plane",
	Long:  `The create command creates a sub-project within a minder control plane.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return cli.MessageAndError("Error binding flags", err)
		}
		return nil
	},
	RunE: createCommand,
}

// createCommand is the command for listing projects
func createCommand(cmd *cobra.Command, _ []string) error {
	client, cleanup, err := GetProjectsClient(cmd)
	if err != nil {
		return cli.MessageAndError("Error getting client", err)
	}
	defer cleanup()

	format := viper.GetString("output")
	project := viper.GetString("project")
	name := viper.GetString("name")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	resp, err := client.CreateProject(cmd.Context(), &minderv1.CreateProjectRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
		Name: name,
	})
	if err != nil {
		return cli.MessageAndError("Error creating sub-project", err)
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
		t := table.New(table.Simple, layouts.Default, cmd.OutOrStdout(), []string{"ID", "Name"})
		t.AddRow(resp.Project.ProjectId, resp.Project.Name)
		t.Render()
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	return nil
}

func init() {
	ProjectCmd.AddCommand(projectCreateCmd)

	projectCreateCmd.Flags().StringP("project", "j", "", "The project to create the sub-project within")
	projectCreateCmd.Flags().StringP("name", "n", "", "The name of the project to create")
	// mark as required
	if err := projectCreateCmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}
	projectCreateCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
