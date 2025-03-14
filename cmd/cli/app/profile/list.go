// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List profiles",
	Long:  `The profile list subcommand lets you list profiles within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(listCommand),
}

// listCommand is the profile "list" subcommand
func listCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewProfileServiceClient(conn)

	project := viper.GetString("project")
	format := viper.GetString("output")
	label := viper.GetString("label")

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	resp, err := client.ListProfiles(ctx, &minderv1.ListProfilesRequest{
		Context:     &minderv1.Context{Project: &project},
		LabelFilter: label,
	})
	if err != nil {
		return cli.MessageAndError("Error getting profiles", err)
	}

	switch format {
	case app.JSON:
		out, err := util.GetJsonFromProto(resp)
		if err != nil {
			return fmt.Errorf("error getting json from proto: %w", err)
		}
		cmd.Println(out)
	case app.YAML:
		for _, prof := range resp.GetProfiles() {
			out, err := util.GetYamlFromProto(prof)
			if err != nil {
				return fmt.Errorf("error getting yaml from proto: %w", err)
			}
			cmd.Println("---")
			cmd.Println(out)
		}
	case app.Table:
		settable := NewProfileSettingsTable()
		for _, v := range resp.Profiles {
			RenderProfileSettingsTable(v, settable)
		}
		settable.Render()
		return nil
	}
	// this is unreachable
	return nil
}

func init() {
	ProfileCmd.AddCommand(listCmd)
	listCmd.Flags().StringP("label", "l", "", "Profile label to filter on")
	if err := listCmd.Flags().MarkHidden("label"); err != nil {
		listCmd.Printf("Error hiding flag: %s", err)
		os.Exit(1)
	}

	// Flags
	listCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
