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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"

	"github.com/stacklok/mediator/internal/engine"
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
		files, err := cmd.Flags().GetStringArray("file")
		if err != nil {
			return fmt.Errorf("error getting file flag: %w", err)
		}

		if err := validateFilesArg(files); err != nil {
			return fmt.Errorf("error validating file arg: %w", err)
		}

		conn, err := util.GrpcForCommand(cmd)
		if err != nil {
			return fmt.Errorf("error getting grpc connection: %w", err)
		}
		defer conn.Close()

		client := pb.NewPolicyServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		for _, f := range files {
			preader, closer, err := util.OpenFileArg(f, cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("error opening file arg: %w", err)
			}
			defer closer()

			r, err := engine.ParseRuleType(preader)
			if err != nil {
				return fmt.Errorf("error parsing rule type: %w", err)
			}

			// create a rule
			resp, err := client.CreateRuleType(ctx, &pb.CreateRuleTypeRequest{
				RuleType: r,
			})
			if err != nil {
				return fmt.Errorf("error creating rule type: %w", err)
			}
			out, err := util.GetJsonFromProto(resp)
			if err != nil {
				return fmt.Errorf("error getting json from proto: %w", err)
			}

			fmt.Println(out)
		}

		return nil
	},
}

func validateFilesArg(files []string) error {
	if files == nil {
		return fmt.Errorf("error: file must be set")
	}

	if slices.Contains(files, "") {
		return fmt.Errorf("error: file must be set")
	}

	if slices.Contains(files, "-") && len(files) > 1 {
		return fmt.Errorf("error: cannot use stdin with other files")
	}

	return nil
}

func init() {
	ruleTypeCmd.AddCommand(RuleType_createCmd)
	RuleType_createCmd.Flags().StringArrayP("file", "f", []string{}, "Path to the YAML defining the rule type (or - for stdin)")
}
