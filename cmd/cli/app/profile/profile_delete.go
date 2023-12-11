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
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var profile_deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a profile within a minder control plane",
	Long: `The minder profile delete subcommand lets you delete profiles within a
minder control plane.`,
	RunE: cli.GRPCClientWrapRunE(func(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
		// delete the profile via GRPC
		id := viper.GetString("id")
		provider := viper.GetString("provider")

		client := pb.NewProfileServiceClient(conn)

		_, err := client.DeleteProfile(ctx, &pb.DeleteProfileRequest{
			Context: &pb.Context{
				Provider: &provider,
			},
			Id: id,
		})
		if err != nil {
			return cli.MessageAndError(cmd, "Error deleting profile", err)
		}

		cmd.Println("Successfully deleted profile with id:", id)

		return nil
	}),
}

func init() {
	ProfileCmd.AddCommand(profile_deleteCmd)
	profile_deleteCmd.Flags().StringP("id", "i", "", "ID of profile to delete")
	profile_deleteCmd.Flags().StringP("provider", "p", "github", "Provider for the profile")
	if err := profile_deleteCmd.MarkFlagRequired("id"); err != nil {
		fmt.Printf("Error marking flag as required: %s", err)
		os.Exit(1)
	}
	// TODO: add a flag for the profile name
}
