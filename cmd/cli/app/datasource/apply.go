// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package datasource

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// applyCmd represents the datasource apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a data source",
	Long:  `The datasource apply subcommand lets you create or update data sources for a project within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(applyCommand),
}

func init() {
	DataSourceCmd.AddCommand(applyCmd)
	// Flags
	applyCmd.Flags().StringArrayP("file", "f", []string{},
		"Path to the YAML defining the data source (or - for stdin). Can be specified multiple times. Can be a directory.")
	// Required
	if err := applyCmd.MarkFlagRequired("file"); err != nil {
		applyCmd.Printf("Error marking flag required: %s", err)
		os.Exit(1)
	}
}

// applyCommand is the datasource apply subcommand
func applyCommand(_ context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewDataSourceServiceClient(conn)

	project := viper.GetString("project")

	fileFlag, err := cmd.Flags().GetStringArray("file")
	if err != nil {
		return cli.MessageAndError("Error parsing file flag", err)
	}

	if err = validateFilesArg(fileFlag); err != nil {
		return cli.MessageAndError("Error validating file flag", err)
	}

	files, err := util.ExpandFileArgs(fileFlag...)
	if err != nil {
		return cli.MessageAndError("Error expanding file args", err)
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	table := initializeTableForList()

	applyFunc := func(ctx context.Context, fileName string, ds *minderv1.DataSource) (*minderv1.DataSource, error) {
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

	for _, f := range files {
		if f.Path != "-" && shouldSkipFile(f.Path) {
			continue
		}
		// cmd.Context() is the root context. We need to create a new context for each file
		// so we can avoid the timeout.
		if err = executeOnOneDataSource(cmd.Context(), table, f.Path, os.Stdin, project, applyFunc); err != nil {
			if f.Expanded && minderv1.YouMayHaveTheWrongResource(err) {
				cmd.PrintErrf("Skipping file %s: not a data source\n", f.Path)
				// We'll skip the file if it's not a data source
				continue
			}
			return cli.MessageAndError(fmt.Sprintf("error applying data source from %s", f.Path), err)
		}
	}
	// Render the table
	table.Render()
	return nil
}
