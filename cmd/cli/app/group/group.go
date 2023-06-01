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

// Package group contains the group logic for the control plane
package group

import (
	"github.com/stacklok/mediator/cmd/cli/app"

	"github.com/spf13/cobra"
)

// GroupCmd is the root command for the group subcommands
var GroupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage groups within a mediator control plane",
	Long: `The medctl group commands allow the management of groups within a 
mediator control plane.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("group called")
	},
}

func init() {
	app.RootCmd.AddCommand(GroupCmd)
}
