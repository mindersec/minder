// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/mindersec/minder/cmd/cli/app"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/util/cli"
)

// whoamiCmd represents the whoami command
var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "whoami for current user",
	Long:  `whoami gets information about the current user from the minder server`,
	RunE:  cli.GRPCClientWrapRunE(whoamiCommand),
}

// whoamiCommand is the whoami subcommand
func whoamiCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewUserServiceClient(conn)

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	userInfo, err := client.GetUser(ctx, &minderv1.GetUserRequest{})
	if err != nil {
		return cli.MessageAndError("Error getting information for user", err)
	}

	renderUserInfoWhoami(conn.Target(), cmd.OutOrStderr(), viper.GetString("output"), userInfo)
	return nil
}

func init() {
	AuthCmd.AddCommand(whoamiCmd)

	whoamiCmd.Flags().StringP("output", "o", app.Table,
		fmt.Sprintf("Output format (one of %s)", strings.Join(app.SupportedOutputFormats(), ",")))
}
