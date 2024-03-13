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

package profile

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// createCmd represents the profile create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a profile",
	Long:  `The profile create subcommand lets you create new profiles for a project within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(createCommand),
}

// createCommand is the profile create subcommand
func createCommand(_ context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	client := minderv1.NewProfileServiceClient(conn)

	provider := viper.GetString("provider")
	project := viper.GetString("project")
	f := viper.GetString("file")
	enableAlerts := viper.GetBool("enable-alerts")
	enableRems := viper.GetBool("enable-remediations")

	onOverride := "on"

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	table := NewProfileTable()

	createFunc := func(ctx context.Context, _ string, p *minderv1.Profile) (*minderv1.Profile, error) {
		if enableAlerts {
			p.Alert = &onOverride
		}
		if enableRems {
			p.Remediate = &onOverride
		}

		// create a profile
		resp, err := client.CreateProfile(ctx, &minderv1.CreateProfileRequest{
			Profile: p,
		})
		if err != nil {
			return nil, err
		}

		return resp.GetProfile(), nil
	}
	// cmd.Context() is the root context. We need to create a new context for each file
	// so we can avoid the timeout.
	profile, err := ExecOnOneProfile(cmd.Context(), table, f, cmd.InOrStdin(), project, provider, createFunc)
	if err != nil {
		return cli.MessageAndError(fmt.Sprintf("error creating profile from %s", f), err)
	}

	// display the name above the table
	cmd.Println("Successfully created new profile named:", profile.GetName())
	table.Render()
	return nil
}

func init() {
	ProfileCmd.AddCommand(createCmd)
	// Flags
	createCmd.Flags().StringP("file", "f", "", "Path to the YAML defining the profile (or - for stdin)")
	createCmd.Flags().Bool("enable-alerts", false, "Explicitly enable alerts for this profile. Overrides the YAML file.")
	createCmd.Flags().Bool("enable-remediations", false, "Explicitly enable remediations for this profile. Overrides the YAML file.")
	// Required
	if err := createCmd.MarkFlagRequired("file"); err != nil {
		createCmd.Printf("Error marking flag required: %s", err)
		os.Exit(1)
	}
}
