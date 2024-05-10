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

// Package image provides the root command for the image subcommands
package image

import "github.com/spf13/cobra"

// CmdImage is the root command for the container subcommands
func CmdImage() *cobra.Command {
	var rtCmd = &cobra.Command{
		Use:   "image",
		Short: "image provides utilities to test minder container image support",
	}

	rtCmd.AddCommand(CmdVerify())
	rtCmd.AddCommand(CmdList())
	rtCmd.AddCommand(CmdListTags())

	return rtCmd
}
