// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package artifact

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	Short: "List artifacts from a provider",
	Long:  `The artifact list subcommand will list artifacts from a provider.`,
	RunE:  cli.GRPCClientWrapRunE(listCommand),
}

// listCommand is the artifact list subcommand
func listCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewArtifactServiceClient(conn)

	provider := viper.GetString("provider")
	project := viper.GetString("project")
	format := viper.GetString("output")
	fromFilter := viper.GetString("from")

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	artifactList, err := client.ListArtifacts(ctx, &minderv1.ListArtifactsRequest{
		Context: &minderv1.Context{Provider: &provider, Project: &project},
		From:    fromFilter,
	},
	)

	if err != nil {
		return cli.MessageAndError("Couldn't list artifacts", err)
	}

	switch format {
	case app.Table:
		t := table.New(table.Simple, layouts.Default,
			[]string{"ID", "Type", "Owner", "Name", "Repository", "Visibility", "Creation date"})
		for _, artifact := range artifactList.Results {
			t.AddRow(
				artifact.ArtifactPk,
				artifact.Type,
				artifact.GetOwner(),
				artifact.GetName(),
				artifact.Repository,
				artifact.Visibility,
				artifact.CreatedAt.AsTime().Format(time.RFC3339),
			)

		}
		t.Render()
	case app.JSON:
		out, err := util.GetJsonFromProto(artifactList)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(artifactList)
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	}

	return nil
}

func init() {
	ArtifactCmd.AddCommand(listCmd)
	// Flags
	listCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	listCmd.Flags().String("from", "", "Filter artifacts from a source, example: from=repository=owner/repo")
}
