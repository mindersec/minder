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

package group

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var group_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a group within a mediator control plane",
	Long: `The medctl group create subcommand lets you create new groups within
a mediator control plane.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("group create called")
	},
}

func init() {
	GroupCmd.AddCommand(group_createCmd)
	group_createCmd.PersistentFlags().StringP("name", "n", "", "Name of the group")
	group_createCmd.PersistentFlags().BoolP("active", "a", true, "Whether the group is active or not")
	if err := viper.BindPFlags(group_createCmd.PersistentFlags()); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
	}
}
