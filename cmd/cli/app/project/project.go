// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package project is the root command for the project subcommands
package project

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// ProjectCmd is the root command for the project subcommands
var ProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage project within a minder control plane",
	Long:  `The minder project commands manage projects within a minder control plane.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(ProjectCmd)
}

// GetProjectsClient is a helper to get the ProjectsServiceClient, supporting mocks via the command context
func GetProjectsClient(cmd *cobra.Command) (minderv1.ProjectsServiceClient, func(), error) {
	ctx, cancel := cli.GetAppContext(cmd.Context(), viper.GetViper())
	cmd.SetContext(ctx)

	if mockClient, ok := cli.GetRPCClient[minderv1.ProjectsServiceClient](ctx); ok {
		return mockClient, func() { cancel() }, nil
	}

	conn, err := cli.GrpcForCommand(cmd, viper.GetViper())
	if err != nil {
		cancel()
		return nil, nil, err
	}

	client := minderv1.NewProjectsServiceClient(conn)

	return client, func() {
		cancel()
		_ = conn.Close()
	}, nil
}
