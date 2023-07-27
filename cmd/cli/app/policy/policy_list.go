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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package policy

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

var policy_listCmd = &cobra.Command{
	Use:   "list",
	Short: "List policies within a mediator control plane",
	Long: `The medic policy list subcommand lets you list policies within a
mediator control plane for an specific group.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)
		format := viper.GetString("output")

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		if err != nil {
			return fmt.Errorf("error getting grpc connection: %w", err)
		}
		defer conn.Close()

		client := pb.NewPolicyServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		provider := viper.GetString("provider")

		if format != "json" && format != "yaml" {
			fmt.Fprintf(os.Stderr, "Error: invalid format: %s\n", format)
		}

		resp, err := client.ListPolicies(ctx, &pb.ListPoliciesRequest{
			Context: &pb.Context{
				Provider: provider,
				// TODO set up group if specified
				// Currently it's inferred from the authorization token
			},
		})
		if err != nil {
			return fmt.Errorf("error getting policies: %w", err)
		}

		if format == "json" {
			output, err := json.MarshalIndent(resp.Policies, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshalling json: %w", err)
			}
			fmt.Println(string(output))
		} else if format == "yaml" {
			enc := yaml.NewEncoder(os.Stdout)
			enc.SetIndent(2)

			if err := enc.Encode(resp.Policies); err != nil {
				return fmt.Errorf("error marshalling yaml: %w", err)
			}
		}

		// this is unreachable
		return nil
	},
}

func init() {
	PolicyCmd.AddCommand(policy_listCmd)
	policy_listCmd.Flags().StringP("provider", "p", "", "Provider to list policies for")
	policy_listCmd.Flags().StringP("output", "o", "yaml", "Output format (json or yaml)")
	// TODO: Take group ID into account
	// policy_listCmd.Flags().Int32P("group-id", "g", 0, "group id to list roles for")

	if err := policy_listCmd.MarkFlagRequired("provider"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
}
