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

package policy_status

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/util"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

var policystatus_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get policy status within a mediator control plane",
	Long: `The medic policy_status get subcommand lets you get policy status within a
mediator control plane for an specific provider/group or policy id and repo-id.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		if err != nil {
			return fmt.Errorf("error getting grpc connection: %w", err)
		}
		defer conn.Close()

		client := pb.NewPolicyServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		provider := viper.GetString("provider")
		group := viper.GetString("group")
		policy_id := viper.GetInt32("policy")
		repo_id := viper.GetInt32("repo")
		format := viper.GetString("output")
		all := viper.GetBool("all")

		if format != "json" && format != "yaml" {
			return fmt.Errorf("error: invalid format: %s", format)
		}

		if provider == "" {
			return fmt.Errorf("provider must be set")
		}

		// at least one of policy_id, repo-id or group needs to be set
		if policy_id == 0 {
			return fmt.Errorf("policy-id must be set")
		}

		if repo_id == 0 {
			return fmt.Errorf("repo-id must be set")
		}

		req := &pb.GetPolicyStatusByIdRequest{
			Context: &pb.Context{
				Provider: provider,
			},
			PolicyId: policy_id,
			RepoId:   repo_id,
			All:      all,
		}

		if group != "" {
			req.Context.Group = &group
		}

		resp, err := client.GetPolicyStatusById(ctx, req)
		if err != nil {
			return fmt.Errorf("error getting policy status: %w", err)
		}

		if format == "json" {
			out, err := util.GetJsonFromProto(resp)
			util.ExitNicelyOnError(err, "Error getting json from proto")
			fmt.Println(out)
		} else {
			out, err := util.GetYamlFromProto(resp)
			util.ExitNicelyOnError(err, "Error getting yaml from proto")
			fmt.Println(out)
		}

		return nil
	},
}

func init() {
	PolicyStatusCmd.AddCommand(policystatus_getCmd)
	policystatus_getCmd.Flags().StringP("provider", "p", "github", "Provider to get policy status for")
	policystatus_getCmd.Flags().StringP("group", "g", "", "group id to get policy status for")
	policystatus_getCmd.Flags().Int32P("policy", "i", 0, "policy id to get policy status for")
	policystatus_getCmd.Flags().Int32P("repo", "r", 0, "repo id to get policy status for")
	policystatus_getCmd.Flags().StringP("output", "o", "yaml", "Output format (json or yaml)")
}
