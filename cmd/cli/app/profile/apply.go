// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package profile

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/internal/util"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// applyCmd represents the profile apply command
var applyCmd = &cobra.Command{
	Use:   "apply [files]",
	Short: "Create or update a profile",
	Long:  `The profile apply subcommand lets you create or update new profiles for a project within Minder.`,
	Args: func(cmd *cobra.Command, args []string) error {
		fileFlag, err := cmd.Flags().GetStringArray("file")
		if err != nil {
			return cli.MessageAndError("Error parsing file flag", err)
		}

		if len(fileFlag) == 0 && len(args) == 0 {
			return fmt.Errorf("no files specified: use positional arguments or the -f flag")
		}
		return nil
	},
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %s", err)
		}
		return nil
	},
	RunE: applyCommand,
}

// applyCommand is the profile apply subcommand
func applyCommand(cmd *cobra.Command, args []string) error {
	fileFlag, _ := cmd.Flags().GetStringArray("file")

	// Combine positional args with -f flag values
	allFiles := append(fileFlag, args...)

	files, err := util.ExpandFileArgs(allFiles...)
	if err != nil {
		return cli.MessageAndError("Error expanding file args", err)
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// getProfileClient handles the context setup and mock injection
	client, closeConn, err := GetProfileClient(cmd)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer closeConn()

	project := viper.GetString("project")

	table := NewProfileRulesTable(cmd.OutOrStdout())

	var failedFiles []string

	applyFunc := func(ctx context.Context, _ string, p *minderv1.Profile) (*minderv1.Profile, error) {
		// create a profile
		resp, err := client.CreateProfile(ctx, &minderv1.CreateProfileRequest{
			Profile: p,
		})

		if err == nil {
			cmd.Printf("Successfully created new profile named: %s\n", p.GetName())
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
		updateResp, err := client.UpdateProfile(ctx, &minderv1.UpdateProfileRequest{
			Profile: p,
		})
		if err != nil {
			return nil, err
		}

		cmd.Printf("Successfully updated existing profile named: %s\n", p.GetName())
		return updateResp.GetProfile(), nil
	}

	for _, f := range files {
		if f.Path != "-" && !cli.IsYAMLFileAndNotATest(f.Path) {
			continue
		}

		if _, err = ExecOnOneProfile(cmd.Context(), table, f.Path, os.Stdin, project, applyFunc); err != nil {
			if f.Expanded && minderv1.YouMayHaveTheWrongResource(err) {
				cmd.PrintErrf("Skipping file %s: not a profile\n", f.Path)
				continue
			}
			cmd.PrintErrln(cli.MessageAndError(fmt.Sprintf("error applying profile from %s", f.Path), err))
			failedFiles = append(failedFiles, f.Path)
			continue
		}
	}

	// Render the combined table for all profiles that were processed
	table.Render()
	if len(failedFiles) > 0 {
		failedList := strings.Join(failedFiles, "\n  ")

		return cli.MessageAndError(
			"failed to apply the following files",
			fmt.Errorf("\n  %s", failedList),
		)
	}
	return nil
}

func init() {
	ProfileCmd.AddCommand(applyCmd)
	// Flags
	applyCmd.Flags().StringArrayP("file", "f", []string{},
		"Path to the YAML defining the profile (or - for stdin). Can be specified multiple files")
}
