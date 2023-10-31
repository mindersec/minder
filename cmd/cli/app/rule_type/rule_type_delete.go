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

	"github.com/stacklok/mediator/internal/util"
	minderv1 "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
)

var ruleType_deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a rule type within a minder control plane",
	Long: `The minder rule type delete subcommand lets you delete profiles within a
minder control plane.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// delete the profile via GRPC
		id := viper.GetString("id")

		conn, err := util.GrpcForCommand(cmd, viper.GetViper())

		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := minderv1.NewProfileServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		_, err = client.DeleteRuleType(ctx, &minderv1.DeleteRuleTypeRequest{
			Context: &minderv1.Context{},
			Id:      id,
		})

		util.ExitNicelyOnError(err, "Error deleting profile")
		cmd.Println("Successfully deleted profile with id:", id)
	},
}

func init() {
	ruleTypeCmd.AddCommand(ruleType_deleteCmd)
	ruleType_deleteCmd.Flags().StringP("id", "i", "", "ID of rule type to delete")
	err := ruleType_deleteCmd.MarkFlagRequired("id")
	util.ExitNicelyOnError(err, "Error marking flag as required")
}
