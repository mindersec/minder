// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package role is the root command for the role subcommands
package role

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app/project"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// RoleCmd is the root command for the project subcommands
var RoleCmd = &cobra.Command{
	Use:   "role",
	Short: "Manage roles within a minder control plane",
	Long:  `The minder role commands manage permissions within a minder control plane.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	project.ProjectCmd.AddCommand(RoleCmd)
	RoleCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}

// GetPermissionsClient is a helper to get the PermissionsServiceClient, supporting mocks
func GetPermissionsClient(cmd *cobra.Command) (minderv1.PermissionsServiceClient, func(), error) {
	ctx, cancel := cli.GetAppContext(cmd.Context(), viper.GetViper())
	cmd.SetContext(ctx)

	if mockClient, ok := cli.GetRPCClient[minderv1.PermissionsServiceClient](ctx); ok {
		return mockClient, func() { cancel() }, nil
	}

	conn, err := cli.GrpcForCommand(cmd, viper.GetViper())
	if err != nil {
		cancel()
		return nil, nil, err
	}

	client := minderv1.NewPermissionsServiceClient(conn)

	return client, func() {
		cancel()
		_ = conn.Close()
	}, nil
}
