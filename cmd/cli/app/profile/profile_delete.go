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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
)

var profile_deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a profile within a mediator control plane",
	Long: `The medic profile delete subcommand lets you delete profiles within a
mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// delete the profile via GRPC
		id := viper.GetString("id")
		provider := viper.GetString("provider")

		conn, err := util.GrpcForCommand(cmd)

		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewProfileServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		_, err = client.DeleteProfile(ctx, &pb.DeleteProfileRequest{
			Context: &pb.Context{
				Provider: provider,
			},
			Id: id,
		})

		util.ExitNicelyOnError(err, "Error deleting profile")
		cmd.Println("Successfully deleted profile with id:", id)
	},
}

func init() {
	ProfileCmd.AddCommand(profile_deleteCmd)
	profile_deleteCmd.Flags().StringP("id", "i", "", "ID of profile to delete")
	profile_deleteCmd.Flags().StringP("provider", "p", "github", "Provider for the profile")
	err := profile_deleteCmd.MarkFlagRequired("id")
	util.ExitNicelyOnError(err, "Error marking flag as required")
	// TODO: add a flag for the profile name
}
