// Copyright 2024 Stacklok, Inc.
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

package app

import (
	"github.com/spf13/cobra"
)

// encryptionCmd groups together the encryption-related commands
var encryptionCmd = &cobra.Command{
	Use:   "encryption",
	Short: "Tools for managing encryption keys",
	Long:  `Use with rotate to re-encrypt provider access tokens with new keys/algorithms`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	RootCmd.AddCommand(encryptionCmd)
	encryptionCmd.PersistentFlags().BoolP("yes", "y", false, "Answer yes to all questions")
}
