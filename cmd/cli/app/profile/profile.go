// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package profile provides the CLI subcommand for managing profiles
package profile

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// ProfileCmd is the root command for the profile subcommands
var ProfileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage profiles",
	Long:  `The profile subcommands allows the management of profiles within Minder.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(ProfileCmd)
	// Flags for all subcommands
	ProfileCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}

// GetProfileClient is a helper to get the ProfileServiceClient (allows mock injection)
func GetProfileClient(cmd *cobra.Command) (minderv1.ProfileServiceClient, func(), error) {
	ctx, cancel := cli.GetAppContext(cmd.Context(), viper.GetViper())
	cmd.SetContext(ctx)

	if mockClient, ok := cli.GetRPCClient[minderv1.ProfileServiceClient](ctx); ok {
		return mockClient, func() { cancel() }, nil
	}

	conn, err := cli.GrpcForCommand(cmd, viper.GetViper())
	if err != nil {
		cancel()
		return nil, nil, err
	}

	client := minderv1.NewProfileServiceClient(conn)

	return client, func() {
		cancel()
		_ = conn.Close()
	}, nil
}
