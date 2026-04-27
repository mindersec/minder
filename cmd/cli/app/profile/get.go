// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get details for a profile",
	Long:  `The profile get subcommand lets you retrieve details for a profile within Minder.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %s", err)
		}
		return nil
	},
	RunE: getCommand,
}

// getCommand is the profile "get" subcommand
func getCommand(cmd *cobra.Command, _ []string) error {
	project := viper.GetString("project")
	format := viper.GetString("output")
	id := viper.GetString("id")
	name := viper.GetString("name")

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}
	if id == "" && name == "" {
		return cli.MessageAndError("Error getting profile", fmt.Errorf("id or name required"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// 3. NOW SETUP GRPC
	client, closeConn, err := GetProfileClient(cmd)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer closeConn()

	var resp protoreflect.ProtoMessage
	var prof *minderv1.Profile
	if id != "" {
		p, err := client.GetProfileById(cmd.Context(), &minderv1.GetProfileByIdRequest{
			Context: &minderv1.Context{Project: &project},
			Id:      id,
		})
		if err != nil {
			return cli.MessageAndError("Error getting profile", err)
		}
		resp = p
		prof = p.GetProfile()
	} else {
		p, err := client.GetProfileByName(cmd.Context(), &minderv1.GetProfileByNameRequest{
			Context: &minderv1.Context{Project: &project},
			Name:    name,
		})
		if err != nil {
			return cli.MessageAndError("Error getting profile", err)
		}
		resp = p
		prof = p.GetProfile()
	}

	switch format {
	case app.YAML:
		out, err := util.GetYamlFromProto(prof)
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	case app.JSON:
		out, err := util.GetJsonFromProto(resp)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.Table:
		settable := NewProfileSettingsTable(cmd.OutOrStdout())
		RenderProfileSettingsTable(prof, settable)
		settable.Render()
		cmd.Println()
		table := NewProfileRulesTable(cmd.OutOrStdout())
		table.SeparateRows()
		RenderProfileRulesTable(prof, table)
		table.Render()
	}
	return nil
}

func init() {
	ProfileCmd.AddCommand(getCmd)
	// Flags
	getCmd.Flags().StringP("id", "i", "", "ID for the profile to query")
	getCmd.Flags().StringP("name", "n", "", "Name for the profile to query")
	getCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	getCmd.MarkFlagsMutuallyExclusive("id", "name")
}
