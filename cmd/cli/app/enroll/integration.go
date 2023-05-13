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

package enroll

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var enroll_integrationCmd = &cobra.Command{
	Use:   "integration",
	Short: "medctl enroll integration",
	Long: `The medctl enroll integration subcommand group lets you enroll new 
integrations within the mediator controlplane.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("enroll integration called")
	},
}

func init() {
	EnrollCmd.AddCommand(enroll_integrationCmd)
	if err := viper.BindPFlags(enroll_integrationCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}
