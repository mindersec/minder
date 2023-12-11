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

package ruletype

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var ruleType_deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a rule type",
	Long: `The minder rule type delete subcommand lets you delete rule types within a
minder control plane.`,
	RunE: cli.GRPCClientWrapRunE(func(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
		// Delete the rule type via GRPC
		id := viper.GetString("id")
		deleteAll := viper.GetBool("all")
		yesFlag := util.GetConfigValue(viper.GetViper(), "yes", "yes", cmd, false).(bool)

		// If id is set, deleteAll cannot be set
		if id != "" && deleteAll {
			return errors.New("cannot set both id and deleteAll")
		}

		// Either name or deleteAll needs to be set
		if id == "" && !deleteAll {
			return errors.New("Either id or deleteAll needs to be set")
		}

		if deleteAll && !yesFlag {
			// Ask for confirmation if deleteAll is set on purpose
			yes := cli.PrintYesNoPrompt(cmd,
				"You are about to permanently delete all of your rule types.",
				"Are you sure?",
				"Delete all rule types operation cancelled.",
				false)
			if !yes {
				return nil
			}
		}
		client := minderv1.NewProfileServiceClient(conn)

		// List of rule types to delete
		rulesToDelete := []*minderv1.RuleType{}
		if !deleteAll {
			// Fetch the rule type from the DB, so we can get its name
			provider := viper.GetString("provider")
			rtype, err := client.GetRuleTypeById(ctx, &minderv1.GetRuleTypeByIdRequest{
				Context: &minderv1.Context{
					Provider: provider,
					// TODO set up project if specified
					// Currently it's inferred from the authorization token
				},
				Id: id,
			})
			if err != nil {
				return fmt.Errorf("Error getting rule type: %w", err)
			}
			// Add the rule type for deletion
			rulesToDelete = append(rulesToDelete, rtype.RuleType)
		} else {
			// List all rule types
			provider := viper.GetString("provider")
			resp, err := client.ListRuleTypes(ctx, &minderv1.ListRuleTypesRequest{
				Context: &minderv1.Context{
					Provider: provider,
					// TODO set up project if specified
					// Currently it's inferred from the authorization token
				},
			})
			if err != nil {
				return fmt.Errorf("Error listing rule types: %w", err)
			}
			rulesToDelete = append(rulesToDelete, resp.RuleTypes...)
		}
		deletedRuleTypes := []string{}
		remainingRuleTypes := []string{}
		// Delete the rule types set for deletion
		for _, ruleType := range rulesToDelete {
			_, err := client.DeleteRuleType(ctx, &minderv1.DeleteRuleTypeRequest{
				Context: &minderv1.Context{},
				Id:      ruleType.GetId(),
			})
			if err != nil {
				remainingRuleTypes = append(remainingRuleTypes, ruleType.GetName())
				continue
			}
			deletedRuleTypes = append(deletedRuleTypes, ruleType.GetName())
		}

		// Print the results
		if len(deletedRuleTypes) == 0 && len(remainingRuleTypes) == 0 {
			cmd.Println("There are no rule types to delete")
			return nil
		}
		if len(deletedRuleTypes) > 0 {
			cmd.Println("\nThe following rule type(s) were successfully deleted:")
			for _, ruleType := range deletedRuleTypes {
				cmd.Println(ruleType)
			}
		}
		if len(remainingRuleTypes) > 0 {
			cmd.Println("\nThe following rule type(s) are referenced by existing profiles and were not deleted:")
			for _, ruleType := range remainingRuleTypes {
				cmd.Println(ruleType)
			}
		}

		return nil
	}),
}

func init() {
	ruleTypeCmd.AddCommand(ruleType_deleteCmd)
	ruleType_deleteCmd.Flags().StringP("provider", "p", "github", "Provider to list rule types for")
	ruleType_deleteCmd.Flags().StringP("id", "i", "", "ID of rule type to delete")
	ruleType_deleteCmd.Flags().BoolP("all", "a", false, "Warning: Deletes all rule types")
	ruleType_deleteCmd.Flags().BoolP("yes", "y", false, "Bypass yes/no prompt when deleting all rule types")
}
