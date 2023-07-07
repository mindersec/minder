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
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/v1"
	"github.com/stacklok/mediator/pkg/util"
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
	Run: func(cmd *cobra.Command, args []string) {
		provider := util.GetConfigValue("provider", "provider", cmd, "")
		group := viper.GetInt32("group-id")
		policyType := util.GetConfigValue("type", "type", cmd, "").(string)
		f := util.GetConfigValue("file", "file", cmd, "").(string)

		var data []byte
		var err error

		if f == "-" {
			data, err = io.ReadAll(os.Stdin)
			util.ExitNicelyOnError(err, "Error reading from stdin")
		} else {
			f = filepath.Clean(f)
			data, err = os.ReadFile(f)
			util.ExitNicelyOnError(err, "Error reading file")
		}

		grpc_host := util.GetConfigValue("grpc_server.host", "grpc-host", cmd, "").(string)
		grpc_port := util.GetConfigValue("grpc_server.port", "grpc-port", cmd, 0).(int)

		conn, err := util.GetGrpcConnection(grpc_host, grpc_port)
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewPolicyServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		policyTypes, err := client.GetPolicyTypes(ctx, &pb.GetPolicyTypesRequest{Provider: provider.(string)})
		util.ExitNicelyOnError(err, "Error getting policy types")

		// check if the policy type is valid
		found := false
		validTypes := make([]string, len(policyTypes.PolicyTypes))
		for _, t := range policyTypes.PolicyTypes {
			validTypes = append(validTypes, t.PolicyType)
			if policyType == t.PolicyType {
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "Invalid policy type - valid policy types are: %v\n", validTypes)
			os.Exit(1)
		}

		// create a policy
		resp, err := client.CreatePolicy(ctx, &pb.CreatePolicyRequest{
			Provider:         provider.(string),
			GroupId:          group,
			Type:             policyType,
			PolicyDefinition: string(data),
		})
		util.ExitNicelyOnError(err, "Error creating policy")

		pol, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			cmd.Println("Created policy: ", resp.Policy.Id)
		} else {
			cmd.Println("Created policy:", string(pol))
		}

	},
}

func init() {
	PolicyCmd.AddCommand(Policy_createCmd)
	Policy_createCmd.Flags().StringP("provider", "n", "", "Provider (github)")
	Policy_createCmd.Flags().Int32P("group-id", "g", 0, "ID of the group to where the policy belongs")
	Policy_createCmd.Flags().StringP("type", "t", "", `Type of policy - must be one valid policy type.
	Please check valid policy types with: medic policy_types list command`)
	Policy_createCmd.Flags().StringP("file", "f", "", "Path to the YAML defining the policy (or - for stdin)")

	if err := Policy_createCmd.MarkFlagRequired("file"); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
	}

	if err := Policy_createCmd.MarkFlagRequired("provider"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
	if err := Policy_createCmd.MarkFlagRequired("type"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}
	if err := Policy_createCmd.MarkFlagRequired("file"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
		os.Exit(1)
	}

}
