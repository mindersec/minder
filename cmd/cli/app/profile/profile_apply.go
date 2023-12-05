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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/internal/util"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// Profile_applyCmd represents the profile apply command
var Profile_applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Create or update a profile within a minder control plane",
	Long: `The minder profile apply subcommand lets you create or update new profiles for a project
within a minder control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		f := util.GetConfigValue(viper.GetViper(), "file", "file", cmd, "").(string)
		proj := viper.GetString("project")

		conn, err := util.GrpcForCommand(cmd, viper.GetViper())
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewProfileServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		table := InitializeTable(cmd)

		applyFunc := func(f string, p *pb.Profile) (*pb.Profile, error) {
			// create a profile
			resp, err := client.CreateProfile(ctx, &pb.CreateProfileRequest{
				Profile: p,
			})
			if err == nil {
				return resp.GetProfile(), nil
			}

			st, ok := status.FromError(err)
			if !ok {
				// We can't parse the error, so just return it
				return nil, fmt.Errorf("error creating rule type from %s: %w", f, err)
			}

			if st.Code() != codes.AlreadyExists {
				return nil, fmt.Errorf("error creating rule type from %s: %w", f, err)
			}

			updateResp, err := client.UpdateProfile(ctx, &pb.UpdateProfileRequest{
				Profile: p,
			})
			if err != nil {
				return nil, fmt.Errorf("error updating rule type from %s: %w", f, err)
			}

			return updateResp.GetProfile(), nil
		}

		if err := execOnOneProfile(table, f, cmd.InOrStdin(), proj, applyFunc); err != nil {
			return err
		}

		table.Render()
		return nil
	},
}

func init() {
	ProfileCmd.AddCommand(Profile_applyCmd)
	Profile_applyCmd.Flags().StringP("file", "f", "", "Path to the YAML defining the profile (or - for stdin)")
	if err := Profile_applyCmd.MarkFlagRequired("file"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag required: %s\n", err)
		os.Exit(1)
	}
	Profile_applyCmd.Flags().StringP("project", "p", "", "Project to create the profile in")
}
