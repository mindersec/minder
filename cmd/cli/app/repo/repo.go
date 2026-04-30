// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package repo contains the repo logic for the control plane
package repo

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// RepoCmd is the root command for the repo subcommands
var RepoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories within a Minder project",
	Long: `Manage repositories within a Minder project.

This command allows you to list, add, and manage repositories
connected to Minder for security analysis and policy enforcement.`,
	Example: `
  # List repositories
    minder repo list

  # Register a repository
    minder repo register --name my-repo --provider github

  # Delete a repository
    minder repo delete --name my-repo
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(RepoCmd)
	// Flags for all subcommands
	RepoCmd.PersistentFlags().StringP("provider", "p", "", "Name of the provider, i.e. github")
	RepoCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}

// getRepoClient is a helper to get the RepositoryServiceClient
func getRepoClient(cmd *cobra.Command) (minderv1.RepositoryServiceClient, func(), error) {
	ctx, cancel := cli.GetAppContext(cmd.Context(), viper.GetViper())
	cmd.SetContext(ctx)

	if mockClient, ok := cli.GetRPCClient[minderv1.RepositoryServiceClient](ctx); ok {
		return mockClient, func() { cancel() }, nil
	}

	conn, err := cli.GrpcForCommand(cmd, viper.GetViper())
	if err != nil {
		cancel()
		return nil, nil, err
	}

	client := minderv1.NewRepositoryServiceClient(conn)

	return client, func() {
		cancel()
		_ = conn.Close()
	}, nil
}
