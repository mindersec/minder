// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package provider is the root command for the provider subcommands
package provider

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// ProviderCmd is the root command for the provider subcommands
var ProviderCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage providers within a minder control plane",
	Long:  `The minder provider commands manage providers within a minder control plane.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(ProviderCmd)
	// Flags for all subcommands
	ProviderCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
	// TODO: get rid of this
	ProviderCmd.PersistentFlags().StringP("provider", "p", "", "DEPRECATED - use `class` flag of `enroll` instead")
	if err := ProviderCmd.PersistentFlags().MarkHidden("provider"); err != nil {
		ProviderCmd.Printf("Error binding flag: %s", err)
		os.Exit(1)
	}
}

func getImplementsAsStrings(p *minderv1.Provider) []string {
	if p == nil {
		return nil
	}

	var impls []string
	for _, impl := range p.GetImplements() {
		i := impl.ToString()
		if i != "" {
			impls = append(impls, i)
		}
	}

	return impls
}

func getAuthFlowsAsStrings(p *minderv1.Provider) []string {
	if p == nil {
		return nil
	}

	var afs []string
	for _, impl := range p.GetAuthFlows() {
		i := impl.ToString()
		if i != "" {
			afs = append(afs, i)
		}
	}

	return afs
}

// GetProviderClient is a helper to get the ProvidersServiceClient, supporting mocks via the command context
func GetProviderClient(cmd *cobra.Command) (minderv1.ProvidersServiceClient, func(), error) {
	ctx, cancel := cli.GetAppContext(cmd.Context(), viper.GetViper())
	cmd.SetContext(ctx)

	if mockClient, ok := cli.GetRPCClient[minderv1.ProvidersServiceClient](ctx); ok {
		return mockClient, func() { cancel() }, nil
	}

	conn, err := cli.GrpcForCommand(cmd, viper.GetViper())
	if err != nil {
		cancel()
		return nil, nil, err
	}

	client := minderv1.NewProvidersServiceClient(conn)

	return client, func() {
		cancel()
		_ = conn.Close()
	}, nil
}
