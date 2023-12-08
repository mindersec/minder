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
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	pb "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// Profile_updateCmd represents the profile update command
var Profile_updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a profile within a minder control plane",
	Long: `The minder profile update subcommand lets you update profiles for a project
within a minder control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: cli.GRPCClientWrapRunE(func(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
		f := util.GetConfigValue(viper.GetViper(), "file", "file", cmd, "").(string)
		proj := viper.GetString("project")

		var err error

		var preader io.Reader

		if f == "" {
			return fmt.Errorf("error: file must be set")
		}

		if f == "-" {
			preader = os.Stdin
		} else {
			f = filepath.Clean(f)
			fopen, err := os.Open(f)
			if err != nil {
				return fmt.Errorf("error opening file: %w", err)
			}

			defer fopen.Close()

			preader = fopen
		}

		client := pb.NewProfileServiceClient(conn)

		p, err := engine.ParseYAML(preader)
		if err != nil {
			return fmt.Errorf("error reading profile from file: %w", err)
		}

		if proj != "" {
			if p.Context == nil {
				p.Context = &pb.Context{}
			}

			p.Context.Project = &proj
		}

		// update a profile
		resp, err := client.UpdateProfile(ctx, &pb.UpdateProfileRequest{
			Profile: p,
		})
		if err != nil {
			return fmt.Errorf("error updating profile: %w", err)
		}

		table := InitializeTable(cmd)
		RenderProfileTable(resp.GetProfile(), table)
		table.Render()
		return nil
	}),
}

func init() {
	ProfileCmd.AddCommand(Profile_updateCmd)
	Profile_updateCmd.Flags().StringP("file", "f", "", "Path to the YAML defining the profile (or - for stdin)")
	Profile_updateCmd.Flags().StringP("project", "p", "", "Project to update the profile in")
}
