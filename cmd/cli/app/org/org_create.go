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
	"github.com/stacklok/mediator/internal/organisation"
	"github.com/stacklok/mediator/pkg/db"
	"github.com/stacklok/mediator/pkg/util"
)

var org_createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an organization within a mediator control plane",
	Long: `The medctl org create subcommand lets you create new organizations
within a mediator control plane.`,
	Run: func(cmd *cobra.Command, args []string) {
		// call the code for creating an organization
		dbConn, err := util.GetDbConnection(cmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting database connection: %s\n", err)
			os.Exit(1)
		}
		store := db.NewStore(dbConn)

		// create the organisation
		name := util.GetConfigValue("name", "name", cmd, "")
		company := util.GetConfigValue("company", "company", cmd, "")
		org, err := organisation.CreateOrganisation(cmd.Context(), store, name.(string), company.(string))

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating organisation: %s\n", err)
			os.Exit(1)
		}
		fmt.Println("Created organisation:", org.Name)

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
