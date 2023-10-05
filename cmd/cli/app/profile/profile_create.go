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
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/engine"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/mediator/v1"
)

// Profile_createCmd represents the profile create command
var Profile_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a profile within a mediator control plane",
	Long: `The medic profile create subcommand lets you create new profiles for a project
within a mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		f := util.GetConfigValue("file", "file", cmd, "").(string)
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

		conn, err := util.GrpcForCommand(cmd)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewProfileServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

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

		// create a profile
		resp, err := client.CreateProfile(ctx, &pb.CreateProfileRequest{
			Profile: p,
		})
		if err != nil {
			return fmt.Errorf("error creating profile: %w", err)
		}

		table := initializeTable(cmd)
		renderProfileTable(resp.GetProfile(), table)
		table.Render()
		return nil
	},
}

func init() {
	ProfileCmd.AddCommand(Profile_createCmd)
	Profile_createCmd.Flags().StringP("file", "f", "", "Path to the YAML defining the profile (or - for stdin)")
	Profile_createCmd.Flags().StringP("project", "p", "", "Project to create the profile in")
}
