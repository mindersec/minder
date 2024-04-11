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

// Package provider is the root command for the provider subcommands
package provider

import (
	"github.com/spf13/cobra"

	"github.com/stacklok/minder/cmd/cli/app"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// ProviderCmd is the root command for the provider subcommands
var ProviderCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage providers within a minder control plane",
	Long:  `The minder provider commands manage providers within a minder control plane.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Usage()
	},
}

func init() {
	app.RootCmd.AddCommand(ProviderCmd)
	// Flags for all subcommands
	ProviderCmd.PersistentFlags().StringP("project", "j", "", "ID of the project")
	// TODO: get rid of this
	ProviderCmd.PersistentFlags().StringP("provider", "p", "", "DEPRECATED - use `class` flag of `enroll` instead")
}

func getImplementsAsStrings(p *minderv1.Provider) []string {
	if p == nil {
		return nil
	}

	var impls []string
	for _, impl := range p.GetImplements() {
		i := impl.ToString()
		if i != "" {
			impls = append(impls, i)
		}
	}

	return impls
}

func getAuthFlowsAsStrings(p *minderv1.Provider) []string {
	if p == nil {
		return nil
	}

	var afs []string
	for _, impl := range p.GetAuthFlows() {
		i := impl.ToString()
		if i != "" {
			afs = append(afs, i)
		}
	}

	return afs
}
