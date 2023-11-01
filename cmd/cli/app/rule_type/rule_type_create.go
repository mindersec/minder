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
	"io"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"

	"github.com/stacklok/mediator/internal/util"
	minderv1 "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
)

// RuleType_createCmd represents the profile create command
var RuleType_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a rule type within a minder control plane",
	Long: `The minder rule type create subcommand lets you create new profiles for a project
within a minder control plane.`,
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

		conn, err := util.GrpcForCommand(cmd, viper.GetViper())
		if err != nil {
			return fmt.Errorf("error getting grpc connection: %w", err)
		}
		defer conn.Close()

		client := minderv1.NewProfileServiceClient(conn)

		expfiles, err := util.ExpandFileArgs(files)
		if err != nil {
			return fmt.Errorf("error expanding file args: %w", err)
		}

		table := initializeTable(cmd)

		for _, f := range expfiles {
			if err := createOneRuleType(client, table, f, os.Stdin); err != nil {
				return fmt.Errorf("error creating rule type: %w", err)
			}
		}

		table.Render()

		return nil
	},
}

func createOneRuleType(
	client minderv1.ProfileServiceClient,
	table *tablewriter.Table,
	f string,
	dashOpen io.Reader,
) error {
	ctx, cancel := util.GetAppContext()
	defer cancel()

	preader, closer, err := util.OpenFileArg(f, dashOpen)
	if err != nil {
		return fmt.Errorf("error opening file arg: %w", err)
	}
	defer closer()

	r, err := minderv1.ParseRuleType(preader)
	if err != nil {
		return fmt.Errorf("error parsing rule type: %w", err)
	}

	// create a rule
	resp, err := client.CreateRuleType(ctx, &minderv1.CreateRuleTypeRequest{
		RuleType: r,
	})
	if err != nil {
		return fmt.Errorf("error creating rule type: %w", err)
	}

	renderRuleTypeTable(resp.RuleType, table)
	return nil
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
	RuleType_createCmd.Flags().StringArrayP("file", "f", []string{},
		"Path to the YAML defining the rule type (or - for stdin). Can be specified multiple times. Can be a directory.")
}
