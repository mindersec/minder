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

package auth

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// auth_logoutCmd represents the logout command
var auth_logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from mediator control plane.",
	Long:  `Logout from mediator control plane. Credentials will be removed from $XDG_CONFIG_HOME/mediator/credentials.json`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Get the XDG_CONFIG_HOME environment variable
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")

		// just delete token from credentials file
		if xdgConfigHome == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
				os.Exit(1)
			}
			xdgConfigHome = filepath.Join(homeDir, ".config")
		}

		filePath := filepath.Join(xdgConfigHome, "mediator", "credentials.json")
		err := os.Remove(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error removing credentials file: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("User logged out.")

	},
}

func init() {
	AuthCmd.AddCommand(auth_logoutCmd)

}
