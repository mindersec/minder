// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package ruletype provides the CLI subcommand for managing rules
package ruletype

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// ruleTypeCmd is the root command for the rule subcommands
var ruleTypeCmd = &cobra.Command{
	Use:   "ruletype",
	Short: "Manage rule types",
	Long:  `The ruletype subcommands allows the management of rule types within Minder.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

// getRuleTypeClient returns the RuleTypeServiceClient, a cleanup function to close the connection and an error
func getRuleTypeClient(cmd *cobra.Command) (minderv1.RuleTypeServiceClient, func(), error) {
	ctx, cancel := cli.GetAppContext(cmd.Context(), viper.GetViper())
	cmd.SetContext(ctx)

	// Check the backpack. Are we running inside a test?
	if mockClient, ok := cli.GetRPCClient[minderv1.RuleTypeServiceClient](ctx); ok {
		return mockClient, func() { cancel() }, nil
	}

	// The backpack is empty. We are in production, make a real connection.
	conn, err := cli.GrpcForCommand(cmd, viper.GetViper())
	if err != nil {
		cancel()
		return nil, nil, err
	}

	client := minderv1.NewRuleTypeServiceClient(conn)

	// Return the client and the closer so the subcommand can manage the lifecycle
	return client, func() {
		cancel()
		_ = conn.Close()
	}, nil
}

func init() {
	app.RootCmd.AddCommand(ruleTypeCmd)
	// Flags for all subcommands
	ruleTypeCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}
