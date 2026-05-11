// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package entity

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register an entity",
	Long: `The entity register subcommand is used to register a new entity instance within Minder.

Identifying properties are specified as key=value pairs using the --property flag.
For example, for a GitHub repository:
  --property github/repo_owner=myorg --property github/repo_name=myrepo`,
	Example: `
  # Register a GitHub repository
    minder entity register --type repository --property github/repo_owner=myorg --property github/repo_name=myrepo
`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}
		return nil
	},
	RunE: registerCommand,
}

// registerCommand is the entity register subcommand
func registerCommand(cmd *cobra.Command, _ []string) error {
	client, closeConn, err := cli.GetCLIClient(cmd, minderv1.NewEntityInstanceServiceClient)
	if err != nil {
		return cli.MessageAndError("Error creating gRPC client", err)
	}
	defer closeConn()

	project := viper.GetString("project")
	provider := viper.GetString("provider")
	format := viper.GetString("output")
	entityTypeStr := viper.GetString("type")
	properties := viper.GetStringSlice("property")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	entityType := minderv1.EntityFromString(entityTypeStr)
	if entityType == minderv1.Entity_ENTITY_UNSPECIFIED {
		return fmt.Errorf("invalid or unspecified entity type %q", entityTypeStr)
	}

	// Parse key=value property pairs
	identifyingProps := make(map[string]*structpb.Value, len(properties))
	for _, prop := range properties {
		key, value, found := strings.Cut(prop, "=")
		if !found {
			return fmt.Errorf("invalid property %q: expected key=value format", prop)
		}
		identifyingProps[key] = structpb.NewStringValue(value)
	}

	resp, err := client.RegisterEntity(cmd.Context(), &minderv1.RegisterEntityRequest{
		Context: &minderv1.ContextV2{
			ProjectId: project,
			Provider:  provider,
		},
		EntityType:            entityType,
		IdentifyingProperties: identifyingProps,
	})
	if err != nil {
		return cli.MessageAndError("Error registering entity", err)
	}

	switch format {
	case app.JSON:
		out, err := util.GetJsonFromProto(resp.GetEntity())
		if err != nil {
			return cli.MessageAndError("Error getting json from proto", err)
		}
		cmd.Println(out)
	case app.YAML:
		out, err := util.GetYamlFromProto(resp.GetEntity())
		if err != nil {
			return cli.MessageAndError("Error getting yaml from proto", err)
		}
		cmd.Println(out)
	default:
		cmd.Printf("Successfully registered entity with ID: %s\n", resp.GetEntity().GetId())
	}

	return nil
}

func init() {
	EntityCmd.AddCommand(registerCmd)
	// Flags
	registerCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
	registerCmd.Flags().StringP("type", "t", "", "Type of entity to register (e.g. repository, artifact, pull_request)")
	registerCmd.Flags().StringArrayP("property", "P", nil, "Identifying property in key=value format (may be repeated)")
	// Required
	if err := registerCmd.MarkFlagRequired("type"); err != nil {
		panic(err)
	}
}
