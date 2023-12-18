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

// Package apply provides the profile apply subcommand for the minder CLI
package apply

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/cmd/cli/app/profile"
	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// applyCmd represents the profile apply command
var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Create or update a profile",
	Long:  `The profile apply subcommand lets you create or update new profiles for a project within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(applyCommand),
}

// applyCommand is the profile apply subcommand
func applyCommand(_ context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	client := minderv1.NewProfileServiceClient(conn)

	provider := viper.GetString("provider")
	project := viper.GetString("project")
	f := viper.GetString("file")

	// Ensure provider is supported
	if !app.IsProviderSupported(provider) {
		return cli.MessageAndError(fmt.Sprintf("Provider %s is not supported yet", provider), fmt.Errorf("invalid argument"))
	}

	table := profile.NewProfileTable()

	applyFunc := func(ctx context.Context, f string, p *minderv1.Profile) (*minderv1.Profile, error) {
		// create a profile
		resp, err := client.CreateProfile(ctx, &minderv1.CreateProfileRequest{
			Context: &minderv1.Context{Provider: &provider, Project: &project},
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
		updateResp, err := client.UpdateProfile(ctx, &minderv1.UpdateProfileRequest{
			Context: &minderv1.Context{Provider: &provider, Project: &project},
			Profile: p,
		})
		if err != nil {
			return nil, err
		}

		return updateResp.GetProfile(), nil
	}
	// cmd.Context() is the root context. We need to create a new context for each file
	// so we can avoid the timeout.
	if err := profile.ExecOnOneProfile(cmd.Context(), table, f, cmd.InOrStdin(), project, applyFunc); err != nil {
		return cli.MessageAndError(fmt.Sprintf("error applying profile from %s", f), err)
	}

	table.Render()
	return nil
}

func init() {
	profile.ProfileCmd.AddCommand(applyCmd)
	// Flags
	applyCmd.Flags().StringP("file", "f", "", "Path to the YAML defining the profile (or - for stdin)")
	// Required
	if err := applyCmd.MarkFlagRequired("file"); err != nil {
		applyCmd.Printf("Error marking flag required: %s", err)
		os.Exit(1)
	}
}
