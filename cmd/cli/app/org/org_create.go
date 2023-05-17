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

package org

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var org_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an organization within a mediator control plane",
	Long: `The medctl org create subcommand lets you create new organizations
within a mediator control plane.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("group create called")
	},
}

func init() {
	OrgCmd.AddCommand(org_createCmd)
	org_createCmd.Flags().StringP("name", "n", "", "Name of the organization")
	org_createCmd.Flags().StringP("company", "c", "", "Company name of the organization")
	if err := viper.BindPFlags(org_createCmd.PersistentFlags()); err != nil {
		fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
	}
}
