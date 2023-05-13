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

package list

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var org_listCmd = &cobra.Command{
	Use:   "org",
	Short: "medctl org commands",
	Long: `The medctl list org subcommand lets you list orgs or their metadata
within the mediator controlplane.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("org called")
	},
}

func init() {
	ListCmd.AddCommand(org_listCmd)
	if err := viper.BindPFlags(org_listCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}
