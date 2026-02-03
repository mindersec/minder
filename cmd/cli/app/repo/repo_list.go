// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package repo

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
	Short: "List repositories",
	Long:  `The repo list subcommand is used to list registered repositories within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(listCommand),
}

// listCommand is the repo list subcommand
func listCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
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
		t := table.New(table.Simple, layouts.Default,
			[]string{"Owner", "Name", "Provider", "Upstream ID"})
		for _, v := range resp.Results {
			t.AddRow(
				v.GetOwner(),
				v.GetName(),
				v.GetContext().GetProvider(),
				fmt.Sprintf("%d", v.GetRepoId()),
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
