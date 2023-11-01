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
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stacklok/mediator/internal/util"
	"github.com/stacklok/mediator/internal/util/cli"
	minderv1 "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
)

var ruleType_deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a rule type",
	Long: `The minder rule type delete subcommand lets you delete rule types within a
minder control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Delete the rule type via GRPC
		id := viper.GetString("id")
		deleteAll := viper.GetBool("all")

		// If id is set, deleteAll cannot be set
		if id != "" && deleteAll {
			fmt.Fprintf(os.Stderr, "Cannot set both id and deleteAll")
			return
		}

		// Either name or deleteAll needs to be set
		if id == "" && !deleteAll {
			fmt.Fprintf(os.Stderr, "Either id or deleteAll needs to be set")
			return
		}

		// Ask for confirmation if deleteAll is set on purpose
		if deleteAll {
			reader := bufio.NewReader(os.Stdin)

			fmt.Print("Warning: Are you sure you want to permanently delete all of your rule types? (yes/no): ")
			response, _ := reader.ReadString('\n')

			response = strings.ToLower(strings.TrimSpace(response))
			if response != "yes" && response != "y" {
				cli.PrintCmd(cmd, cli.Header.Render("Delete account operation cancelled."))
				return
			}
		}
		// Create GRPC connection
		conn, err := util.GrpcForCommand(cmd, viper.GetViper())
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := minderv1.NewProfileServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		// Delete the rule type(s)
		resp, err := client.DeleteRuleType(ctx, &minderv1.DeleteRuleTypeRequest{
			Context: &minderv1.Context{
				Provider: viper.GetString("provider"),
			},
			Id:        id,
			DeleteAll: deleteAll,
		})
		util.ExitNicelyOnError(err, "Error deleting rule type(s)")

		// Print the results
		if len(resp.DeletedRuleTypes) == 0 && len(resp.RemainingRuleTypes) == 0 {
			cmd.Println("There are no rule types to delete")
			return
		}
		if len(resp.DeletedRuleTypes) > 0 {
			cmd.Println("The following rule type(s) were successfully deleted:")
			for _, ruleType := range resp.DeletedRuleTypes {
				cmd.Println(ruleType)
			}
		}
		if len(resp.RemainingRuleTypes) > 0 {
			cmd.Println("The following rule type(s) are referenced by existing profiles and were not deleted:")
			for _, ruleType := range resp.RemainingRuleTypes {
				cmd.Println(ruleType)
			}
		}
	},
}

func init() {
	ruleTypeCmd.AddCommand(ruleType_deleteCmd)
	ruleType_deleteCmd.Flags().StringP("provider", "p", "", "Provider to list rule types for")
	ruleType_deleteCmd.Flags().StringP("id", "i", "", "ID of rule type to delete")
	ruleType_deleteCmd.Flags().BoolP("all", "a", false, "Warning: Deletes all rule types")
	err := ruleType_deleteCmd.MarkFlagRequired("provider")
	util.ExitNicelyOnError(err, "Error marking flag as required")
}
