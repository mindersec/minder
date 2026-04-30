// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package artifact provides the artifact subcommands
package artifact

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// ArtifactCmd is the artifact subcommand
var ArtifactCmd = &cobra.Command{
	Use:   "artifact",
	Short: "Manage artifacts within a minder control plane",
	Long:  `The minder artifact commands allow the management of artifacts within a minder control plane`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(ArtifactCmd)
	// Flags for all subcommands
	ArtifactCmd.PersistentFlags().StringP("provider", "p", "", "Name of the provider, i.e. github")
	ArtifactCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}

// getArtifactClient is a helper to get the ArtifactServiceClient
func getArtifactClient(cmd *cobra.Command) (minderv1.ArtifactServiceClient, func(), error) {
	ctx := cmd.Context()
	ctx, cancel := cli.GetAppContext(ctx, viper.GetViper())
	cmd.SetContext(ctx)

	if mockClient, ok := cli.GetRPCClient[minderv1.ArtifactServiceClient](ctx); ok {
		return mockClient, func() { cancel() }, nil
	}

	conn, err := cli.GrpcForCommand(cmd, viper.GetViper())
	if err != nil {
		cancel()
		return nil, nil, err
	}

	client := minderv1.NewArtifactServiceClient(conn)

	return client, func() {
		cancel()
		_ = conn.Close()
	}, nil
}
