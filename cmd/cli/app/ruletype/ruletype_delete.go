// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package ruletype

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

type ruleTypeBlock struct {
	Name     string
	Profiles []string
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a rule type",
	Long:  `The ruletype delete subcommand lets you delete rule types within Minder.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("error binding flags: %w", err)
		}

		id, _ := cmd.Flags().GetString("id")
		name, _ := cmd.Flags().GetString("name")

		if id != "" && name != "" {
			return fmt.Errorf("please provide either the --id or --name flag, but not both")
		}

		return nil
	},
	RunE: deleteCommand,
}

func deleteCommand(cmd *cobra.Command, _ []string) error {
	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	client, closeConn, err := GetRuleTypeClient(cmd)
	if err != nil {
		return cli.MessageAndError("Error connecting to server", err)
	}
	defer closeConn()

	project := viper.GetString("project")
	id := viper.GetString("id")
	name := viper.GetString("name")
	deleteAll := viper.GetBool("all")
	yesFlag := viper.GetBool("yes")

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
		// Fetch the rule type from the DB by either ID or Name
		if id != "" {
			rtype, err := client.GetRuleTypeById(cmd.Context(), &minderv1.GetRuleTypeByIdRequest{
				Context: &minderv1.Context{Project: &project},
				Id:      id,
			})
			if err != nil {
				return cli.MessageAndError("Error getting rule type by id", err)
			}
			rulesToDelete = append(rulesToDelete, rtype.RuleType)
		}

		if name != "" {
			rtype, err := client.GetRuleTypeByName(cmd.Context(), &minderv1.GetRuleTypeByNameRequest{
				Context: &minderv1.Context{Project: &project},
				Name:    name,
			})
			if err != nil {
				return cli.MessageAndError("Error getting rule type by name", err)
			}
			rulesToDelete = append(rulesToDelete, rtype.RuleType)
		}

	} else {
		// List all rule types
		resp, err := client.ListRuleTypes(cmd.Context(), &minderv1.ListRuleTypesRequest{
			Context: &minderv1.Context{Project: &project},
		})
		if err != nil {
			return cli.MessageAndError("Error listing rule types", err)
		}
		rulesToDelete = append(rulesToDelete, resp.RuleTypes...)
	}

	// Delete the rule types set for deletion
	deletedRuleTypes, remainingRuleTypes := deleteRuleTypes(cmd.Context(), client, rulesToDelete, project)

	// Print the results
	printDeleteResults(cmd, deletedRuleTypes, remainingRuleTypes)

	return nil

}

func deleteRuleTypes(
	ctx context.Context,
	client minderv1.RuleTypeServiceClient,
	rulesToDelete []*minderv1.RuleType,
	project string,
) ([]string, []ruleTypeBlock) {
	var deletedRuleTypes []string
	var remainingRuleTypes []ruleTypeBlock

	for _, ruleType := range rulesToDelete {
		_, err := client.DeleteRuleType(ctx, &minderv1.DeleteRuleTypeRequest{
			Context: &minderv1.Context{Project: &project},
			Id:      ruleType.GetId(),
		})
		if err != nil {
			profiles := extractProfiles(err.Error())

			remainingRuleTypes = append(remainingRuleTypes, ruleTypeBlock{
				Name:     ruleType.GetName(),
				Profiles: profiles,
			})
			continue
		}
		deletedRuleTypes = append(deletedRuleTypes, ruleType.GetName())
	}
	return deletedRuleTypes, remainingRuleTypes
}

func printDeleteResults(
	cmd *cobra.Command,
	deletedRuleTypes []string,
	remainingRuleTypes []ruleTypeBlock,
) {
	if len(deletedRuleTypes) == 0 && len(remainingRuleTypes) == 0 {
		cmd.Println("There are no rule types to delete")
		return
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
			cmd.Println(ruleType.Name)
		}

		cmd.Println("\nThey are referenced by profiles:")

		seen := map[string]bool{}
		for _, b := range remainingRuleTypes {
			for _, p := range b.Profiles {
				if !seen[p] {
					cmd.Println(p)
					seen[p] = true
				}
			}
		}
	}
}

func extractProfiles(errMsg string) []string {
	profilesRegex := regexp.MustCompile(`used by profiles (.+)`)
	match := profilesRegex.FindStringSubmatch(strings.ToLower(errMsg))
	if len(match) < 2 {
		return []string{}
	}

	profilesPart := strings.TrimSpace(match[1])

	parts := strings.Split(profilesPart, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	return parts
}

func init() {
	ruleTypeCmd.AddCommand(deleteCmd)
	// Flags
	deleteCmd.Flags().StringP("id", "i", "", "ID of rule type to delete")
	deleteCmd.Flags().StringP("name", "n", "", "Name of rule type to delete")
	deleteCmd.Flags().BoolP("all", "a", false, "Warning: Deletes all rule types")
	deleteCmd.Flags().BoolP("yes", "y", false, "Bypass yes/no prompt when deleting all rule types")
	// Exclusive
	deleteCmd.MarkFlagsOneRequired("id", "name", "all")
	deleteCmd.MarkFlagsMutuallyExclusive("id", "name", "all")
}
