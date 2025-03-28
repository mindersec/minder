// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/pkg/api"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/fileconvert"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply multiple minder resources",
	Long:  `The apply subcommand lets you apply multiple Minder resources at once.`,
	RunE:  cli.GRPCClientWrapRunE(applyCommand),
}

// applyCommand is the general-purpose "apply" subcommand
func applyCommand(ctx context.Context, cmd *cobra.Command, args []string, conn *grpc.ClientConn) error {
	// Step 1: Collect inputs, by reading files or directories.  Use the "-f" flag if set, positional arguments otherwise.
	fileNames := args
	if len(viper.GetStringSlice("file")) > 0 {
		fileNames = viper.GetStringSlice("file")
	}
	objects, err := fileconvert.ResourcesFromPaths(cmd.Printf, fileNames...)
	if err != nil {
		return cli.MessageAndError("Error reading resources", err)
	}

	// Step 2: sort objects by type
	var profiles []*minderv1.Profile
	var ruleTypes []*minderv1.RuleType
	var dataSources []*minderv1.DataSource

	// Explicitly set the project for each resource
	project := viper.GetString("project")
	v1Context := &minderv1.Context{
		Project: &project,
	}
	v2Context := &minderv1.ContextV2{
		ProjectId: project,
	}

	for _, obj := range objects {
		switch rsrc := obj.(type) {
		case *minderv1.Profile:
			rsrc.Context = v1Context
			profiles = append(profiles, rsrc)
		case *minderv1.RuleType:
			rsrc.Context = v1Context
			ruleTypes = append(ruleTypes, rsrc)
		case *minderv1.DataSource:
			rsrc.Context = v2Context
			dataSources = append(dataSources, rsrc)
		default:
			return fmt.Errorf("unsupported object type: %T", obj)
		}
	}

	// Step 3: apply objects, starting with DataSources
	dataSourceClient := minderv1.NewDataSourceServiceClient(conn)
	for _, dataSource := range dataSources {
		if err := api.UpsertDataSource(ctx, dataSourceClient, dataSource); err != nil {
			return cli.MessageAndError(fmt.Sprintf("Unable to create datasource %s", dataSource.Name), err)
		}
	}

	ruleTypeClient := minderv1.NewRuleTypeServiceClient(conn)
	for _, ruleType := range ruleTypes {
		if err := api.UpsertRuleType(ctx, ruleTypeClient, ruleType); err != nil {
			return cli.MessageAndError(fmt.Sprintf("Unable to create ruletype %s", ruleType.Name), err)
		}
	}

	profileClient := minderv1.NewProfileServiceClient(conn)
	for _, profile := range profiles {
		if err := api.UpsertProfile(ctx, profileClient, profile); err != nil {
			return cli.MessageAndError(fmt.Sprintf("Unable to create profile %s", profile.Name), err)
		}
	}

	return nil
}

func init() {
	RootCmd.AddCommand(applyCmd)
	// Flags
	applyCmd.Flags().StringSliceP("file", "f", []string{}, "Input file or directory")
}
