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

package rule_type

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

// RuleType_createCmd represents the policy create command
var RuleType_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a rule type within a mediator control plane",
	Long: `The medic rule type create subcommand lets you create new policies for a group
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

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewPolicyServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		// We transcode to JSON so we can decode it straight to the protobuf structure
		w := &bytes.Buffer{}
		if err := util.TranscodeYAMLToJSON(preader, w); err != nil {
			return fmt.Errorf("error converting yaml to json: %w", err)
		}

		r := &pb.RuleType{}
		if err := json.NewDecoder(w).Decode(r); err != nil {
			return fmt.Errorf("error decoding json: %w", err)
		}

		// create a policy
		resp, err := client.CreateRuleType(ctx, &pb.CreateRuleTypeRequest{
			RuleType: r,
		})
		if err != nil {
			return fmt.Errorf("error creating rule type: %w", err)
		}

		m := protojson.MarshalOptions{
			Indent: "  ",
		}
		out, err := m.Marshal(resp)
		util.ExitNicelyOnError(err, "Error marshalling json")
		fmt.Println(string(out))

		return nil
	},
}

func init() {
	ruleTypeCmd.AddCommand(RuleType_createCmd)
	RuleType_createCmd.Flags().StringP("file", "f", "", "Path to the YAML defining the rule type (or - for stdin)")
}
