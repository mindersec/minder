// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package role

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
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
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}

		format := viper.GetString("output")

		return app.IsOutputFormatSupported(format)
	},
	RunE: listCommand,
}

// listCommand is the command for listing roles
func listCommand(cmd *cobra.Command, _ []string) error {
	project := viper.GetString("project")
	format := viper.GetString("output")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	client, cleanup, err := cli.GetCLIClient(cmd, minderv1.NewPermissionsServiceClient)
	if err != nil {
		return err
	}
	defer cleanup()

	resp, err := client.ListRoles(cmd.Context(), &minderv1.ListRolesRequest{
		Context: &minderv1.Context{
			Project: &project,
		},
	})
	if err != nil {
		return cli.MessageAndError("Error listing roles", err)
	}

	switch format {
	case app.JSON, app.YAML:
		return cli.PrintProto(cmd, format, resp)
	case app.Table:
		t := initializeTableForList(cmd.OutOrStdout())
		for _, r := range resp.Roles {
			t.AddRow(r.Name, r.Description)
		}
		t.Render()
	}
	return nil
}

func initializeTableForList(out io.Writer) table.Table {
	return table.New(table.Simple, layouts.Default, out, []string{"Name", "Description"})
}

func init() {
	RoleCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
