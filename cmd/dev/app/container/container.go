// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package container provides the root command for the container subcommands
package container

import "github.com/spf13/cobra"

// CmdContainer is the root command for the container subcommands
func CmdContainer() *cobra.Command {
	var rtCmd = &cobra.Command{
		Use:   "container",
		Short: "container provides utilities to test minder container support",
	}

	rtCmd.AddCommand(CmdVerify())
	rtCmd.AddCommand(CmdList())
	rtCmd.AddCommand(CmdListTags())

	return rtCmd
}
