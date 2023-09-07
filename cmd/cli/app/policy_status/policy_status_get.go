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

	"github.com/stacklok/mediator/cmd/cli/app"
	"github.com/stacklok/mediator/internal/util"
	"github.com/stacklok/mediator/pkg/entities"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
)

var policystatus_getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get policy status within a mediator control plane",
	Long: `The medic policy_status get subcommand lets you get policy status within a
mediator control plane for an specific provider/group or policy id, entity type and entity id.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := util.GrpcForCommand(cmd)
		if err != nil {
			return fmt.Errorf("error getting grpc connection: %w", err)
		}
		defer conn.Close()

		client := pb.NewPolicyServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		provider := viper.GetString("provider")
		group := viper.GetString("group")
		policyId := viper.GetInt32("policy")
		entityId := viper.GetInt32("entity")
		entityType := viper.GetString("entity-type")
		format := viper.GetString("output")

		switch format {
		case app.JSON, app.YAML, app.Table:
		default:
			return fmt.Errorf("error: invalid format: %s", format)
		}

		if provider == "" {
			return fmt.Errorf("provider must be set")
		}

		if policyId == 0 {
			return fmt.Errorf("policy-id must be set")
		}

		if entityId == 0 {
			return fmt.Errorf("entity-id must be set")
		}

		req := &pb.GetPolicyStatusByIdRequest{
			Context: &pb.Context{
				Provider: provider,
			},
			PolicyId: policyId,
			EntitySelector: &pb.GetPolicyStatusByIdRequest_Entity{
				Entity: &pb.GetPolicyStatusByIdRequest_EntityTypedId{
					Id:   entityId,
					Type: entities.FromString(entityType),
				},
			},
		}

		if group != "" {
			req.Context.Group = &group
		}

		resp, err := client.GetPolicyStatusById(ctx, req)
		if err != nil {
			return fmt.Errorf("error getting policy status: %w", err)
		}

		switch format {
		case app.JSON:
			out, err := util.GetJsonFromProto(resp)
			util.ExitNicelyOnError(err, "Error getting json from proto")
			fmt.Println(out)
		case app.YAML:
			out, err := util.GetYamlFromProto(resp)
			util.ExitNicelyOnError(err, "Error getting yaml from proto")
			fmt.Println(out)
		case app.Table:
			handlePolicyStatusListTable(cmd, resp)
		}

		return nil
	},
}

func init() {
	PolicyStatusCmd.AddCommand(policystatus_getCmd)
	policystatus_getCmd.Flags().StringP("provider", "p", "github", "Provider to get policy status for")
	policystatus_getCmd.Flags().StringP("group", "g", "", "group id to get policy status for")
	policystatus_getCmd.Flags().Int32P("policy", "i", 0, "policy id to get policy status for")
	policystatus_getCmd.Flags().StringP("entity-type", "t", "",
		fmt.Sprintf("the entity type to get policy status for (one of %s)", entities.KnownTypesCSV()))
	policystatus_getCmd.Flags().Int32P("entity", "e", 0, "entity id to get policy status for")
	policystatus_getCmd.Flags().StringP("output", "o", app.Table, "Output format (json, yaml or table)")
}
