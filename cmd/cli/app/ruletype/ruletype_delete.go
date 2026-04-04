// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a rule type",
	Long:  `The ruletype delete subcommand lets you delete rule types within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(deleteCommand),
}

// deleteCommand is the rule type delete subcommand
func deleteCommand(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewRuleTypeServiceClient(conn)

	project := viper.GetString("project")
	id := viper.GetString("id")
	deleteAll := viper.GetBool("all")
	yesFlag := viper.GetBool("yes")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

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

	// List of rule types to delete
	var rulesToDelete []*minderv1.RuleType
	if !deleteAll {
		// Fetch the rule type from the DB, so we can get its name
		rtype, err := client.GetRuleTypeById(ctx, &minderv1.GetRuleTypeByIdRequest{
			Context: &minderv1.Context{Project: &project},
			Id:      id,
		})
		if err != nil {
			return cli.MessageAndError("Error getting rule type", err)
		}
		// Add the rule type for deletion
		rulesToDelete = append(rulesToDelete, rtype.RuleType)
	} else {
		// List all rule types
		resp, err := client.ListRuleTypes(ctx, &minderv1.ListRuleTypesRequest{
			Context: &minderv1.Context{Project: &project},
		})
		if err != nil {
			return cli.MessageAndError("Error listing rule types", err)
		}
		rulesToDelete = append(rulesToDelete, resp.RuleTypes...)
	}

	// Delete the rule types set for deletion
	deletedRuleTypes, remainingRuleTypes, ruleTypeToProfiles := deleteRuleTypes(ctx, client, rulesToDelete, project)

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

		if len(ruleTypeToProfiles) > 0 {
			cmd.Println("\nThey are referenced by profiles:")

			seen := map[string]bool{}
			for _, rt := range remainingRuleTypes {
				for _, p := range ruleTypeToProfiles[rt] {
					if !seen[p] {
						cmd.Println(p)
						seen[p] = true
					}
				}
			}
		}

	}

	return nil
}

func deleteRuleTypes(
	ctx context.Context,
	client minderv1.RuleTypeServiceClient,
	rulesToDelete []*minderv1.RuleType,
	project string,
) ([]string, []string, map[string][]string) {
	var deletedRuleTypes []string
	var remainingRuleTypes []string
	var ruleTypeToProfiles = make(map[string][]string)
	for _, ruleType := range rulesToDelete {
		_, err := client.DeleteRuleType(ctx, &minderv1.DeleteRuleTypeRequest{
			Context: &minderv1.Context{Project: &project},
			Id:      ruleType.GetId(),
		})
		if err != nil {
			remainingRuleTypes = append(remainingRuleTypes, ruleType.GetName())

			errMsg := err.Error()
			if strings.Contains(strings.ToLower(errMsg), "used by profiles") {
				profiles := extractProfiles(errMsg)
				if len(profiles) > 0 {
					ruleTypeToProfiles[ruleType.GetName()] = profiles
				}
			}

			continue
		}
		deletedRuleTypes = append(deletedRuleTypes, ruleType.GetName())
	}
	return deletedRuleTypes, remainingRuleTypes, ruleTypeToProfiles
}

func extractProfiles(errMsg string) []string {
	errMsg = strings.ToLower(errMsg)

	idx := strings.Index(errMsg, "used by profiles ")
	if idx == -1 {
		return nil
	}

	profilesPart := errMsg[idx+len("used by profiles "):]
	profilesPart = strings.TrimSpace(profilesPart)
	profilesPart = strings.TrimSuffix(profilesPart, ".")

	return strings.Split(profilesPart, ", ")
}

func init() {
	ruleTypeCmd.AddCommand(deleteCmd)
	// Flags
	deleteCmd.Flags().StringP("id", "i", "", "ID of rule type to delete")
	deleteCmd.Flags().BoolP("all", "a", false, "Warning: Deletes all rule types")
	deleteCmd.Flags().BoolP("yes", "y", false, "Bypass yes/no prompt when deleting all rule types")
	// TODO: add a flag for the rule type name
	// Exclusive
	deleteCmd.MarkFlagsOneRequired("id", "all")
	deleteCmd.MarkFlagsMutuallyExclusive("id", "all")

}
