// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"cmp"
	"fmt"
	"slices"
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
	Short: "List repositories",
	Long:  `The repo list subcommand is used to list registered repositories within Minder.`,
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

func listCommand(cmd *cobra.Command, _ []string) error {
	provider := viper.GetString("provider")
	project := viper.GetString("project")
	format := viper.GetString("output")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	client, cleanup, err := getRepoClient(cmd)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer cleanup()

	// Paginate through all pages until cursor is empty.
	var allRepos []*minderv1.Repository
	var cursor string
	const maxPaginationRequests = 100
	reqCount := 0
	for reqCount < maxPaginationRequests {

		reqCount++
		resp, err := client.ListRepositories(cmd.Context(), &minderv1.ListRepositoriesRequest{
			Context: &minderv1.Context{Provider: &provider, Project: &project},
			Cursor:  cursor,
		})
		if err != nil {
			return cli.MessageAndError("Error listing repositories", err)
		}
		allRepos = append(allRepos, resp.Results...)
		cursor = resp.Cursor
		if cursor == "" {
			break
		}
	}

	switch format {
	case app.Table:
		t := table.New(table.Simple, layouts.Default, cmd.OutOrStdout(),
			[]string{"Owner", "Name", "Provider", "Upstream ID"}).
			SetAutoMerge(true)

		slices.SortFunc(allRepos, func(a, b *minderv1.Repository) int {
			return cmp.Or(
				strings.Compare(a.GetOwner(), b.GetOwner()),
				strings.Compare(a.GetContext().GetProvider(), b.GetContext().GetProvider()),
			)
		})

		for _, v := range allRepos {
			t.AddRow(
				v.GetOwner(),
				v.GetName(),
				v.GetContext().GetProvider(),
				fmt.Sprintf("%d", v.GetRepoId()),
			)
		}
		t.Render()
	case app.JSON, app.YAML:
		// Build a combined response for JSON/YAML output.
		return cli.PrintProto(cmd, format, &minderv1.ListRepositoriesResponse{Results: allRepos})
	}
	return nil
}

func init() {
	RepoCmd.AddCommand(listCmd)
	// Flags
	listCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
