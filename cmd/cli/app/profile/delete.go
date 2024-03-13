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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a profile",
	Long:  `The profile delete subcommand lets you delete profiles within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(deleteCommand),
}

// deleteCommand is the profile delete subcommand
func deleteCommand(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	client := minderv1.NewProfileServiceClient(conn)

	provider := viper.GetString("provider")
	project := viper.GetString("project")
	id := viper.GetString("id")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// Delete profile
	_, err := client.DeleteProfile(ctx, &minderv1.DeleteProfileRequest{
		Context: &minderv1.Context{Provider: &provider, Project: &project},
		Id:      id,
	})
	if err != nil {
		return cli.MessageAndError("Error deleting profile", err)
	}

	cmd.Println("Successfully deleted profile with id:", id)

	return nil
}

func init() {
	ProfileCmd.AddCommand(deleteCmd)
	// Flags
	deleteCmd.Flags().StringP("id", "i", "", "ID of profile to delete")
	// TODO: add a flag for the profile name
	// Required
	if err := deleteCmd.MarkFlagRequired("id"); err != nil {
		deleteCmd.Printf("Error marking flag required: %s", err)
		os.Exit(1)
	}
}
