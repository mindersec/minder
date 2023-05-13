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

package delete

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var group_deleteCmd = &cobra.Command{
	Use:   "group",
	Short: "medctl group commands",
	Long: `The medctl group subcommand lets you delete new groups within
the mediator controlplane.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("delete group called")
	},
}

func init() {
	DeleteCmd.AddCommand(group_deleteCmd)
	if err := viper.BindPFlags(group_deleteCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}
