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

package policy

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/engine"
	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// Policy_createCmd represents the policy create command
var Policy_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a policy within a mediator control plane",
	Long: `The medic policy create subcommand lets you create new policies for a group
within a mediator control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		f := util.GetConfigValue("file", "file", cmd, "").(string)

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

		client := pb.NewPolicyServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		p, err := engine.ParseYAML(preader)
		if err != nil {
			return fmt.Errorf("error reading fragment from file: %w", err)
		}

		// create a policy
		resp, err := client.CreatePolicy(ctx, &pb.CreatePolicyRequest{
			Policy: p,
		})
		if err != nil {
			return fmt.Errorf("error creating policy: %w", err)
		}

		table := initializeTable(cmd)
		renderPolicyTable(resp.GetPolicy(), table)
		table.Render()
		return nil
	},
}

func init() {
	PolicyCmd.AddCommand(Policy_createCmd)
	Policy_createCmd.Flags().StringP("file", "f", "", "Path to the YAML defining the policy (or - for stdin)")
}
