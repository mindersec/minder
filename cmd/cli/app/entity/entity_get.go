// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity

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

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get entity details",
	Long:  `The entity get subcommand is used to get details for an entity instance within Minder.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}
		return nil
	},
	RunE: getCommand,
}

// getCommand is the entity get subcommand
func getCommand(cmd *cobra.Command, _ []string) error {
	client, closeConn, err := cli.GetCLIClient(cmd, minderv1.NewEntityInstanceServiceClient)
	if err != nil {
		return cli.MessageAndError("Error creating gRPC client", err)
	}
	defer closeConn()

	project := viper.GetString("project")
	provider := viper.GetString("provider")
	format := viper.GetString("output")
	id := viper.GetString("id")
	name := viper.GetString("name")
	entityTypeStr := viper.GetString("type")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	var entity *minderv1.EntityInstance

	if id != "" {
		resp, err := client.GetEntityById(cmd.Context(), &minderv1.GetEntityByIdRequest{
			Context: &minderv1.ContextV2{
				ProjectId: project,
				Provider:  provider,
			},
			Id: id,
		})
		if err != nil {
			return cli.MessageAndError("Error getting entity by ID", err)
		}
		entity = resp.GetEntity()
	} else {
		entityType := minderv1.EntityFromString(entityTypeStr)
		if entityType == minderv1.Entity_ENTITY_UNSPECIFIED {
			return fmt.Errorf("invalid or unspecified entity type %q; required when using --name", entityTypeStr)
		}

		resp, err := client.GetEntityByName(cmd.Context(), &minderv1.GetEntityByNameRequest{
			Context: &minderv1.ContextV2{
				ProjectId: project,
				Provider:  provider,
			},
			Name:       name,
			EntityType: entityType,
		})
		if err != nil {
			return cli.MessageAndError("Error getting entity by name", err)
		}
		entity = resp.GetEntity()
	}

	switch format {
	case app.Table:
		emoji := viper.GetBool("emoji")
		t := table.New(table.Simple, layouts.Default, cmd.OutOrStdout(),
			[]string{"Type", "Name", "Provider", "ID"})
		typeIcon := table.GetEntityTypeIcon(entity.GetType().String(), emoji)
		t.AddRowWithColor(
			typeIcon,
			layouts.NoColor(entity.GetName()),
			layouts.NoColor(entity.GetContext().GetProvider()),
			layouts.NoColor(entity.GetId()),
		)
		t.Render()
	case app.JSON:
		out, err := util.GetJsonFromProto(entity)
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(entity)
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	}

	return nil
}

func init() {
	EntityCmd.AddCommand(getCmd)
	// Flags
	getCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join([]string{app.Table, app.JSON, app.YAML}, ",")))
	getCmd.Flags().StringP("id", "i", "", "ID of the entity to get")
	getCmd.Flags().StringP("name", "n", "", "Name of the entity to get")
	getCmd.Flags().StringP("type", "t", "", "Type of entity (e.g. repository, artifact, pull_request); required with --name")
	getCmd.Flags().Bool("emoji", true, "Use emojis in the output")
	// Require either id or name
	getCmd.MarkFlagsOneRequired("id", "name")
	getCmd.MarkFlagsMutuallyExclusive("id", "name")
}
