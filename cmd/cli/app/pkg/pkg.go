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

// Package pkg contains the pkgs logic for the control plane
package pkg

import (
	"github.com/spf13/cobra"

	"github.com/stacklok/mediator/cmd/cli/app"
)

// PackagesCmd is the root command for the repo subcommands
var PkgCmd = &cobra.Command{
	Use:   "pkg",
	Short: "Manage repositories within a mediator control plane",
	Long: `The medic pkg commands allow the management of repositories within a 
mediator control plane.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(PkgCmd)
}
