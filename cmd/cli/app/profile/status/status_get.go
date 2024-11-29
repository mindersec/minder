// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package status

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

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
	RunE:  cli.GRPCClientWrapRunE(getCommand),
}

// getCommand is the profile "get" subcommand
func getCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewProfileServiceClient(conn)

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

	var resp *minderv1.GetProfileStatusResponse
	var err error

	if profileId != "" {
		resp, err = client.GetProfileStatusById(ctx, &minderv1.GetProfileStatusByIdRequest{
			Context: &minderv1.Context{Project: &project},
			Id:      profileId,
			Entity: &minderv1.EntityTypedId{
				Id:   entityId,
				Type: minderv1.EntityFromString(entityType),
			},
		})
	} else {
		resp, err = client.GetProfileStatusByName(ctx, &minderv1.GetProfileStatusByNameRequest{
			Context: &minderv1.Context{Project: &project},
			Name:    profileName,
			Entity: &minderv1.EntityTypedId{
				Id:   entityId,
				Type: minderv1.EntityFromString(entityType),
			},
		})
	}
	if err != nil {
		return cli.MessageAndError("Error getting profile status", err)
	}

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
		table := profile.NewProfileStatusTable()
		profile.RenderProfileStatusTable(resp.ProfileStatus, table)
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
