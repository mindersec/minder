// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package datasource

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1" // Added import
)

// DataSourceCmd is the root command for the data source subcommands
var DataSourceCmd = &cobra.Command{
	Use:   "datasource",
	Short: "Manage data sources within a minder control plane",
	Long:  "The data source subcommand allows the management of data sources within Minder.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(DataSourceCmd)
	// Flags for all subcommands
	DataSourceCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}

// GetDataSourceClient returns the DataSourceServiceClient, a cleanup function to close the connection and an error
func GetDataSourceClient(cmd *cobra.Command) (minderv1.DataSourceServiceClient, func(), error) {
	ctx, cancel := cli.GetAppContext(cmd.Context(), viper.GetViper())
	cmd.SetContext(ctx)

	if mockClient, ok := cli.GetRPCClient[minderv1.DataSourceServiceClient](ctx); ok {
		return mockClient, func() { cancel() }, nil
	}

	conn, err := cli.GrpcForCommand(cmd, viper.GetViper())
	if err != nil {
		cancel()
		return nil, nil, err
	}

	return minderv1.NewDataSourceServiceClient(conn), func() {
		cancel()
		_ = conn.Close()
	}, nil
}
