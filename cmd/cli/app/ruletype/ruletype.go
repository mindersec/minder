// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package ruletype provides the CLI subcommand for managing rules
package ruletype

import (
	"context"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

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

func getRuleTypeClient(ctx context.Context, conn grpc.ClientConnInterface) minderv1.RuleTypeServiceClient {
	// 1. Check the backpack. Are we running inside a test?
	if mockClient, ok := cli.GetRPCClient[minderv1.RuleTypeServiceClient](ctx); ok {
		return mockClient
	}

	// 2. The backpack is empty. We are in production, make a real connection.
	return minderv1.NewRuleTypeServiceClient(conn)
}

func init() {
	app.RootCmd.AddCommand(ruleTypeCmd)
	// Flags for all subcommands
	ruleTypeCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
}
