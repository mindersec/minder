// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package invite provides the auth invite command for the minder CLI.
package invite

import (
	"context"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// inviteDeclineCmd represents the decline command
var inviteDeclineCmd = &cobra.Command{
	Use:     "decline",
	Short:   "Declines a pending invitation",
	Long:    `Declines a pending invitation for the current minder user`,
	PreRunE: cli.EnsureCredentials,
	RunE:    cli.GRPCClientWrapRunE(inviteDeclineCommand),
	Args:    cobra.ExactArgs(1),
}

// inviteDeclineCommand is the "invite decline" subcommand
func inviteDeclineCommand(ctx context.Context, cmd *cobra.Command, args []string, conn *grpc.ClientConn) error {
	client := minderv1.NewUserServiceClient(conn)
	code := args[0]
	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	res, err := client.ResolveInvitation(ctx, &minderv1.ResolveInvitationRequest{
		Accept: false,
		Code:   code,
	})
	if err != nil {
		return cli.MessageAndError("Error resolving invitation", err)
	}
	cmd.Printf("Invitation %s for %s to become %s of project %s was declined!\n", code, res.Email, res.Role, res.ProjectDisplay)
	return nil
}

func init() {
	inviteCmd.AddCommand(inviteDeclineCmd)
}
