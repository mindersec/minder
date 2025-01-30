// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// applyCmd represents the profile apply command
var applyCmd = &cobra.Command{
	Use:   "apply [file]",
	Short: "Create or update a profile",
	Long:  `The profile apply subcommand lets you create or update new profiles for a project within Minder.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  cli.GRPCClientWrapRunE(applyCommand),
}

// applyCommand is the profile apply subcommand
func applyCommand(_ context.Context, cmd *cobra.Command, args []string, conn *grpc.ClientConn) error {
	client := minderv1.NewProfileServiceClient(conn)

	project := viper.GetString("project")

	// Get file from positional arg if provided, otherwise from -f flag
	var f string
	if len(args) > 0 {
		f = args[0]
	} else {
		f = viper.GetString("file")
	}

	if f == "" {
		return fmt.Errorf("file is required - provide as argument or via --file flag")
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	table := NewProfileTable()
	alreadyExists := false

	applyFunc := func(ctx context.Context, _ string, p *minderv1.Profile) (*minderv1.Profile, error) {
		// create a profile
		resp, err := client.CreateProfile(ctx, &minderv1.CreateProfileRequest{
			Profile: p,
		})
		if err == nil {
			return resp.GetProfile(), nil
		}

		st, ok := status.FromError(err)
		if !ok {
			// We can't parse the error, so just return it
			return nil, err
		}

		if st.Code() != codes.AlreadyExists {
			return nil, err
		}
		// The profile already exists, so update it
		alreadyExists = true
		updateResp, err := client.UpdateProfile(ctx, &minderv1.UpdateProfileRequest{
			Profile: p,
		})
		if err != nil {
			return nil, err
		}

		return updateResp.GetProfile(), nil
	}

	// cmd.Context() is the root context. We need to create a new context for each file
	// so we can avoid the timeout.
	profile, err := ExecOnOneProfile(cmd.Context(), table, f, cmd.InOrStdin(), project, applyFunc)
	if err != nil {
		return cli.MessageAndError(fmt.Sprintf("error applying profile from %s", f), err)
	}

	// display the name above the table
	// use a different message depending on whether this is a new project
	if alreadyExists {
		cmd.Println("Successfully updated existing profile named:", profile.GetName())
	} else {
		cmd.Println("Successfully created new profile named:", profile.GetName())
	}
	table.Render()
	return nil
}

func init() {
	ProfileCmd.AddCommand(applyCmd)
	// Flags
	applyCmd.Flags().StringP("file", "f", "", "Path to the YAML defining the profile (or - for stdin)")
}
