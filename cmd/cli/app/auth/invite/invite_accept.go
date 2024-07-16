//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package invite provides the auth invite command for the minder CLI.
package invite

import (
	"context"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// inviteAcceptCmd represents the accept command
var inviteAcceptCmd = &cobra.Command{
	Use:   "accept",
	Short: "Accept a pending invitation",
	Long:  `Accept a pending invitation for the current minder user`,
	PreRunE: cli.EnsureCredentials,
	RunE:  cli.GRPCClientWrapRunE(inviteAcceptCommand),
	Args:  cobra.ExactArgs(1),
}



// inviteAcceptCommand is the "invite accept" subcommand
func inviteAcceptCommand(ctx context.Context, cmd *cobra.Command, args []string, conn *grpc.ClientConn) error {
	client := minderv1.NewUserServiceClient(conn)
	code := args[0]
	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	res, err := client.ResolveInvitation(ctx, &minderv1.ResolveInvitationRequest{
		Accept: true,
		Code:   code,
	})
	if err != nil {
		return cli.MessageAndError("Error resolving invitation", err)
	}
	cmd.Printf("Invitation %s for %s to become %s of project %s was accepted!\n", code, res.Email, res.Role, res.ProjectDisplay)
	return nil
}

func init() {
	inviteCmd.AddCommand(inviteAcceptCmd)
}
