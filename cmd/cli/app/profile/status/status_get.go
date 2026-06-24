// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/proto"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/cmd/cli/app/profile"
	"github.com/mindersec/minder/internal/engine/entities"
	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get profile status",
	Long:  `The profile status get subcommand lets you get profile status within Minder.`,
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
	profileName := viper.GetString("name")
	profileId := viper.GetString("id")
	entityId := viper.GetString("entity")
	entityType := viper.GetString("entity-type")
	format := viper.GetString("output")

	// Ensure the output format is supported
	if !app.IsOutputFormatSupported(format) {
		return cli.MessageAndError(fmt.Sprintf("Output format %s not supported", format), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	client, closer, err := cli.GetCLIClient(cmd, minderv1.NewProfileServiceClient)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer closer()

	entity := &minderv1.EntityTypedId{
		Type: minderv1.EntityFromString(entityType),
	}
	// If entityId is a UUID, fill the `id` field, otherwise fill the name field.
	if _, err := uuid.Parse(entityId); err == nil {
		entity.Id = entityId
	} else {
		entity.Name = entityId
	}

	if profileId != "" {
		resp, err := getProfileStatusById(cmd.Context(), client, project, profileId, entity)
		if err != nil {
			return cli.MessageAndError("Error getting profile status", err)
		}
		return formatAndDisplayOutput(cmd, format, resp, viper.GetBool("emoji"))
	} else if profileName != "" {
		resp, err := getProfileStatusByName(cmd.Context(), client, project, profileName, entity)
		if err != nil {
			return cli.MessageAndError("Error getting profile status", err)
		}
		return formatAndDisplayOutput(cmd, format, resp, viper.GetBool("emoji"))
	}

	return cli.MessageAndError("Error getting profile status", fmt.Errorf("profile id or profile name required"))
}

func getProfileStatusById(
	ctx context.Context,
	client minderv1.ProfileServiceClient,
	project, profileId string,
	entity *minderv1.EntityTypedId,
) (*minderv1.GetProfileStatusByIdResponse, error) {
	if profileId == "" {
		return nil, cli.MessageAndError("Error getting profile status", fmt.Errorf("profile id required"))
	}

	resp, err := client.GetProfileStatusById(ctx, &minderv1.GetProfileStatusByIdRequest{
		Context: &minderv1.Context{Project: &project},
		Id:      profileId,
		Entity:  entity,
	})
	if err != nil {
		return nil, err
	}

	return &minderv1.GetProfileStatusByIdResponse{
		ProfileStatus:        resp.ProfileStatus,
		RuleEvaluationStatus: resp.RuleEvaluationStatus,
	}, nil
}

func getProfileStatusByName(
	ctx context.Context,
	client minderv1.ProfileServiceClient,
	project, profileName string,
	entity *minderv1.EntityTypedId,
) (*minderv1.GetProfileStatusByNameResponse, error) {
	if profileName == "" {
		return nil, cli.MessageAndError("Error getting profile status", fmt.Errorf("profile name required"))
	}

	resp, err := client.GetProfileStatusByName(ctx, &minderv1.GetProfileStatusByNameRequest{
		Context: &minderv1.Context{Project: &project},
		Name:    profileName,
		Entity:  entity,
	})
	if err != nil {
		return nil, err
	}

	return &minderv1.GetProfileStatusByNameResponse{
		ProfileStatus:        resp.ProfileStatus,
		RuleEvaluationStatus: resp.RuleEvaluationStatus,
	}, nil
}

type protoWithProfileStatus interface {
	proto.Message
	GetProfileStatus() *minderv1.ProfileStatus
}

func formatAndDisplayOutput(
	cmd *cobra.Command, format string, resp protoWithProfileStatus, emoji bool) error {
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
		table := profile.NewProfileStatusTable(cmd.OutOrStdout())
		profile.RenderProfileStatusTable(resp.GetProfileStatus(), table, emoji)
		table.Render()
	}
	return nil
}

func init() {
	profileStatusCmd.AddCommand(getCmd)
	// Flags
	getCmd.Flags().StringP("entity", "e", "", "Entity ID to get profile status for")
	getCmd.Flags().StringP("entity-type", "t", "",
		fmt.Sprintf("the entity type to get profile status for (one of %s)", entities.KnownTypesCSV()))
	getCmd.Flags().StringP("id", "i", "", "ID to get profile status for")
	getCmd.Flags().StringP("name", "n", "", "Profile name to get profile status for")
	getCmd.Flags().Bool("emoji", true, "Use emojis in the output")

	getCmd.MarkFlagsOneRequired("id", "name")
	// Required
	if err := getCmd.MarkFlagRequired("entity"); err != nil {
		getCmd.Printf("Error marking flag required: %s", err)
		os.Exit(1)
	}

	if err := getCmd.MarkFlagRequired("entity-type"); err != nil {
		getCmd.Printf("Error marking flag required: %s", err)
		os.Exit(1)
	}
}
