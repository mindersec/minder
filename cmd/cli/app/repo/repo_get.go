// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package repo

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get repository details",
	Long:  `The repo get subcommand is used to get details for a registered repository within Minder.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}

		format := viper.GetString("output")

		// Ensure the output format is supported
		if !app.IsOutputFormatSupported(format) || format == app.Table {
			return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
		}

		return nil
	},
	RunE: getCommand,
}

// getCommand is the repo get subcommand
func getCommand(cmd *cobra.Command, _ []string) error {
	provider := viper.GetString("provider")
	project := viper.GetString("project")
	format := viper.GetString("output")
	repoid := viper.GetString("id")
	name := viper.GetString("name")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	client, cleanup, err := getRepoClient(cmd)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer cleanup()

	var repository *minderv1.Repository

	// check repo by id
	if repoid != "" {
		resp, err := client.GetRepositoryById(cmd.Context(), &minderv1.GetRepositoryByIdRequest{
			Context:      &minderv1.Context{Provider: &provider, Project: &project},
			RepositoryId: repoid,
		})
		if err != nil {
			return cli.MessageAndError("Error getting repo by id", err)
		}
		repository = resp.Repository
	} else {
		// check repo by name
		resp, err := client.GetRepositoryByName(cmd.Context(), &minderv1.GetRepositoryByNameRequest{
			Context: &minderv1.Context{Provider: &provider, Project: &project},
			Name:    name,
		})
		if err != nil {
			return cli.MessageAndError("Error getting repo by name", err)
		}
		repository = resp.Repository
	}

	// print result just in JSON or YAML
	switch format {
	case app.JSON:
		out, err := util.GetJsonFromProto(repository)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(repository)
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	}

	return nil
}

func init() {
	RepoCmd.AddCommand(getCmd)
	// Flags
	getCmd.Flags().StringP("output", "o", app.JSON,
		fmt.Sprintf("Output format (one of %s)", strings.Join([]string{app.JSON, app.YAML}, ",")))
	getCmd.Flags().StringP("name", "n", "", "Name of the repository (owner/name format)")
	getCmd.Flags().StringP("id", "i", "", "ID of the repo to query")
	// Required
	getCmd.MarkFlagsOneRequired("name", "id")
	getCmd.MarkFlagsMutuallyExclusive("name", "id")
}
