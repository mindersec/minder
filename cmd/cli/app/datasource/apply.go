// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package datasource provides functionalities to manage and apply data sources.
package datasource

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// applyCmd represents the datasource apply command
var applyCmd = &cobra.Command{
	Use:   "apply [files...]",
	Short: "Apply a data source",
	Long:  `The datasource apply subcommand lets you create or update data sources for a project within Minder.`,
	RunE:  applyCommand,
	Args:  cobra.ArbitraryArgs,
}

func init() {
	DataSourceCmd.AddCommand(applyCmd)
	// Flags
	applyCmd.Flags().StringArrayP("file", "f", []string{},
		"Path or HTTP(S) URL to the YAML defining the data source (or - for stdin). Can be specified multiple times. Can be a directory.")
}

func applyCommand(cmd *cobra.Command, args []string) error {
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return fmt.Errorf("error binding flags: %w", err)
	}

	fileFlag, err := cmd.Flags().GetStringArray("file")
	if err != nil {
		return cli.MessageAndError("Error parsing file flag", err)
	}

	// Combine positional args with -f flag values
	allFiles := append(fileFlag, args...)

	if len(allFiles) == 0 {
		return fmt.Errorf("no files specified: use positional arguments or the -f flag")
	}

	if err = validateFilesArg(fileFlag); err != nil {
		return cli.MessageAndError("Error validating file flag", err)
	}

	files, err := util.ExpandFileArgs(allFiles...)
	if err != nil {
		return cli.MessageAndError("Error expanding file args", err)
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	client, cleanup, err := cli.GetCLIClient(cmd, minderv1.NewDataSourceServiceClient)
	if err != nil {
		return err
	}
	defer cleanup()

	project := viper.GetString("project")
	table := initializeTableForList(cmd.OutOrStdout())

	for _, f := range files {
		if f.Path != "-" && shouldSkipFile(f.Path) {
			continue
		}
		if err = executeOnOneDataSource(cmd.Context(), table, f.Path, os.Stdin, project,
			createOrUpdateDataSource(client)); err != nil {
			if f.Expanded && minderv1.YouMayHaveTheWrongResource(err) {
				cmd.PrintErrf("Skipping file %s: not a data source\n", f.Path)
				continue
			}
			return cli.MessageAndError(fmt.Sprintf("error applying data source from %s", f.Path), err)
		}
	}
	table.Render()
	return nil
}

// createOrUpdateDataSource returns a function that creates or updates a data source.
func createOrUpdateDataSource(client minderv1.DataSourceServiceClient) func(
	ctx context.Context, fileName string, ds *minderv1.DataSource,
) (*minderv1.DataSource, error) {
	return func(ctx context.Context, fileName string, ds *minderv1.DataSource) (*minderv1.DataSource, error) {
		createResp, err := client.CreateDataSource(ctx, &minderv1.CreateDataSourceRequest{
			DataSource: ds,
		})

		if err == nil {
			return createResp.DataSource, nil
		}

		st, ok := status.FromError(err)
		if !ok {
			// We can't parse the error, so just return it
			return nil, fmt.Errorf("error creating data source from %s: %w", fileName, err)
		}

		if st.Code() != codes.AlreadyExists {
			return nil, fmt.Errorf("error creating data source from %s: %w", fileName, err)
		}

		updateResp, err := client.UpdateDataSource(ctx, &minderv1.UpdateDataSourceRequest{
			DataSource: ds,
		})

		if err != nil {
			return nil, fmt.Errorf("error updating data source from %s: %w", fileName, err)
		}

		return updateResp.DataSource, nil
	}
}
