// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/pkg/api"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/fileconvert"
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply multiple minder resources",
	Long:  `The apply subcommand lets you apply multiple Minder resources at once.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %s", err)
		}

		return nil

	},
	RunE: applyCommand,
}

// applyCommand is the general-purpose "apply" subcommand
//
//nolint:gocyclo
func applyCommand(cmd *cobra.Command, args []string) error {
	// Step 1: Collect inputs, by reading files or directories.  Use the "-f" flag if set, positional arguments otherwise.
	fileNames := args
	argFiles, _ := cmd.Flags().GetStringSlice("file")
	if len(argFiles) > 0 {
		fileNames = argFiles
	}
	objects, err := fileconvert.ResourcesFromPaths(cmd.Printf, fileNames...)
	if err != nil {
		return cli.MessageAndError("Error reading resources", err)
	}

	if len(objects) == 0 {
		return errors.New("no resources found")
	}
	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

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
	dataSourceClient, dsClose, err := cli.GetCLIClient(cmd, minderv1.NewDataSourceServiceClient)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer dsClose()
	for _, dataSource := range dataSources {
		if err := api.UpsertDataSource(cmd.Context(), dataSourceClient, dataSource); err != nil {
			return cli.MessageAndError(fmt.Sprintf("Unable to create datasource %s", dataSource.Name), err)
		}
	}

	ruleTypeClient, rtClose, err := cli.GetCLIClient(cmd, minderv1.NewRuleTypeServiceClient)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer rtClose()
	for _, ruleType := range ruleTypes {
		if err := api.UpsertRuleType(cmd.Context(), ruleTypeClient, ruleType); err != nil {
			return cli.MessageAndError(fmt.Sprintf("Unable to create ruletype %s", ruleType.Name), err)
		}
	}

	profileClient, pClose, err := cli.GetCLIClient(cmd, minderv1.NewProfileServiceClient)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer pClose()
	for _, profile := range profiles {
		if err := api.UpsertProfile(cmd.Context(), profileClient, profile); err != nil {
			return cli.MessageAndError(fmt.Sprintf("Unable to create profile %s", profile.Name), err)
		}
	}

	return nil
}

func init() {
	RootCmd.AddCommand(applyCmd)
	// Flags
	applyCmd.Flags().StringP("project", "j", "", "ID of the project")
	applyCmd.Flags().StringSliceP("file", "f", []string{}, "Input file or directory")
}
