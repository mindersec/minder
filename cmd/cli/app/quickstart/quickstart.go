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

// Package quickstart provides the quickstart command for the minder CLI
// which is used to provide the means to quickly get started with minder.
package quickstart

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/cmd/cli/app/profile"
	"github.com/stacklok/minder/cmd/cli/app/repo"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

//go:embed embed*
var content embed.FS

var cmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Quickstart minder",
	Long:  "The quickstart command provide the means to quickly get started with minder",
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		proj := viper.GetString("project")
		provider := viper.GetString("provider")
		// Get the grpc connection and other resources
		conn, err := util.GrpcForCommand(cmd, viper.GetViper())
		if err != nil {
			return fmt.Errorf("error getting grpc connection: %w", err)
		}
		defer conn.Close()

		// Prompt to register repositories
		results, msg, err := repo.RegisterCmd(cmd, args)
		util.ExitNicelyOnError(err, msg)

		var registeredRepos []string
		for _, result := range results {
			repo := fmt.Sprintf("%s/%s", result.Repository.Owner, result.Repository.Name)
			registeredRepos = append(registeredRepos, repo)
		}

		// Confirm user wants to proceed with the quickstart process of creating a rule type
		yes := cli.PrintYesNoPrompt(cmd,
			"You are about to create a rule of type - secret_scanning:",
			"Proceed?",
			"Quickstart operation cancelled.")
		if !yes {
			return nil
		}

		// Create a client for the profile and rule type service
		client := minderv1.NewProfileServiceClient(conn)

		// Creating the rule type
		cmd.Println("Creating rule type...")

		// Load the rule type from the embedded file system
		preader, _ := content.Open("embed/secret_scanning.yaml")
		rt, err := minderv1.ParseRuleType(preader)
		if err != nil {
			return fmt.Errorf("error parsing rule type: %w", err)
		}
		if rt.Context == nil {
			rt.Context = &minderv1.Context{}
		}

		if proj != "" {
			rt.Context.Project = &proj
		}

		if provider != "" {
			rt.Context.Provider = provider
		}

		// Create the rule type in minder (new context, so we don't time out)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		_, err = client.CreateRuleType(ctx, &minderv1.CreateRuleTypeRequest{
			RuleType: rt,
		})
		if err != nil {
			if st, ok := status.FromError(err); ok {
				if st.Code() != codes.AlreadyExists {
					return fmt.Errorf("error creating rule type from: %w", err)
				}
				cmd.Println("Rule type secret_scanning already exists")
			} else {
				return fmt.Errorf("error creating rule type from: %w", err)
			}
		}

		// Confirm user wants to proceed with the quickstart process of creating a profile
		yes = cli.PrintYesNoPrompt(cmd,
			fmt.Sprintf(
				"You are about to create a profile in Minder with the following properties:\n\n"+
					"Rules: secret_scanning (enabled)\n\nSelected repositories:\n\n%s",
				strings.Join(registeredRepos[:], "\n"),
			),
			"Proceed?",
			"Quickstart operation cancelled.")
		if !yes {
			return nil
		}

		// Creating the profile
		cmd.Println("Creating profile...")
		preader, _ = content.Open("embed/profile.yaml")

		// Load the profile from the embedded file system
		p, err := engine.ParseYAML(preader)
		if err != nil {
			return fmt.Errorf("error parsing profile: %w", err)
		}

		if p.Context == nil {
			rt.Context = &minderv1.Context{}
		}

		if proj != "" {
			p.Context.Project = &proj
		}

		if provider != "" {
			p.Context.Provider = provider
		}

		// Create the profile in minder (new context, so we don't time out)
		ctx, cancel = util.GetAppContext()
		defer cancel()

		resp, err := client.CreateProfile(ctx, &minderv1.CreateProfileRequest{
			Profile: p,
		})
		if err != nil {
			if st, ok := status.FromError(err); ok {
				if st.Code() != codes.AlreadyExists {
					return fmt.Errorf("error creating profile: %w", err)
				}
				cmd.Println("Profile already exists")
				return nil
			}
			return fmt.Errorf("error creating profile: %w", err)
		}

		table := profile.InitializeTable(cmd)
		profile.RenderProfileTable(resp.GetProfile(), table)
		table.Render()
		return nil
	},
}

func init() {
	app.RootCmd.AddCommand(cmd)
	cmd.Flags().StringP("project", "r", "", "Project to create the quickstart profile in")
	cmd.Flags().StringP("provider", "p", "", "Name of the provider")
	if err := cmd.MarkFlagRequired("provider"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
}
