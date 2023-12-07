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
	minderprov "github.com/stacklok/minder/cmd/cli/app/provider"
	"github.com/stacklok/minder/cmd/cli/app/repo"
	"github.com/stacklok/minder/internal/engine"
	"github.com/stacklok/minder/internal/util"
	"github.com/stacklok/minder/internal/util/cli"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

const (
	// nolint:lll
	stepPromptMsgWelcome = `
Welcome! 👋 

You are about to go through the quickstart process for Minder. Throughout this, you will: 

* Enroll your provider
* Register your repositories
* Create a rule type
* Create a profile

Let's get started!
`
	// nolint:lll
	stepPromptMsgEnroll = `
Step 1 - Enroll your provider.

This will enroll the provider for your repositories.

Currently Minder works with Github, but we are planning support for other providers too!

The command we are about to do is the following:

minder provider enroll --provider github
`
	// nolint:lll
	stepPromptMsgRegister = `
Step 2 - Register your repositories.

Now that you have enrolled your provider successfully, you can register your repositories.

The command we are about to do is the following:

minder repo register --provider github
`
	// nolint:lll
	stepPromptMsgRuleType = `
Step 3 - Create your first rule type - secret_scanning.

Now that you have registered your repositories with Minder, let's create your first rule type!

For the purpose of this quickstart, we are going to use a rule of type "secret_scanning" (secret_scanning.yaml).
Secret scanning is about protecting you from accidentally leaking secrets in your repository.

The command we are about to do is the following:

minder rule_type create -f secret_scanning.yaml
`
	// nolint:lll
	stepPromptMsgProfile = `
Step 4 - Create your first profile.

So far you have enrolled a provider, registered your repositories and created a rule type for secrets scanning.
It's time to stitch all of that together by creating a profile. 

Let's create a profile that enables secret scanning for all of your registered repositories. 

We'll enable the remediate and alert features too, so Minder can automatically remediate any non-compliant repositories and alert you if needed.

Your profile will be applied to the following repositories:

%s

The command we are about to do is the following:

minder profile create -f quickstart-profile.yaml
`
	// nolint:lll
	stepPromptMsgFinish = `
Congratulations! 🎉 You've now successfully created your first profile in Minder!

You can now continue to explore Minder's features by adding or removing more repositories, create custom profiles with various rules, and much more.

For more information about Minder, see:
* GitHub - https://github.com/stacklok/minder
* CLI commands - https://minder-docs.stacklok.dev/ref/cli/minder
* Minder rules & profiles - https://github.com/stacklok/minder-rules-and-profiles
* Official documentation - https://minder-docs.stacklok.dev

Thank you for using Minder!
`
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

		// Confirm user wants to go through the quickstart process
		yes := cli.PrintYesNoPrompt(cmd,
			stepPromptMsgWelcome,
			"Proceed?",
			"Quickstart operation cancelled.",
			true)
		if !yes {
			return nil
		}

		// Step 1 - Confirm enrolling
		yes = cli.PrintYesNoPrompt(cmd,
			stepPromptMsgEnroll,
			"Proceed?",
			"Quickstart operation cancelled.",
			true)
		if !yes {
			return nil
		}

		// Enroll provider
		msg, err := minderprov.EnrollProviderCmd(cmd, args)
		if err != nil {
			return fmt.Errorf("%s: %w", msg, err)
		}

		// Get the grpc connection and other resources
		conn, err := util.GrpcForCommand(cmd, viper.GetViper())
		if err != nil {
			return fmt.Errorf("error getting grpc connection: %w", err)
		}
		defer conn.Close()

		// Step 2 - Confirm repository registration
		yes = cli.PrintYesNoPrompt(cmd,
			stepPromptMsgRegister,
			"Proceed?",
			"Quickstart operation cancelled.",
			true)
		if !yes {
			return nil
		}

		// Prompt to register repositories
		results, msg, err := repo.RegisterCmd(cmd, args)
		util.ExitNicelyOnError(err, msg)

		var registeredRepos []string
		for _, result := range results {
			repo := fmt.Sprintf("%s/%s", result.Repository.Owner, result.Repository.Name)
			registeredRepos = append(registeredRepos, repo)
		}

		// Step 3 - Confirm rule type creation
		yes = cli.PrintYesNoPrompt(cmd,
			stepPromptMsgRuleType,
			"Proceed?",
			"Quickstart operation cancelled.",
			true)
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
			rt.Context.Provider = &provider
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

		// Step 4 - Confirm profile creation
		yes = cli.PrintYesNoPrompt(cmd,
			fmt.Sprintf(stepPromptMsgProfile, strings.Join(registeredRepos[:], "\n")),
			"Proceed?",
			"Quickstart operation cancelled.",
			true)
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
			p.Context = &minderv1.Context{}
		}

		if proj != "" {
			p.Context.Project = &proj
		}

		if provider != "" {
			p.Context.Provider = &provider
		}

		// Create the profile in minder (new context, so we don't time out)
		ctx, cancel = util.GetAppContext()
		defer cancel()

		alreadyExists := ""
		resp, err := client.CreateProfile(ctx, &minderv1.CreateProfileRequest{
			Profile: p,
		})
		if err != nil {
			if st, ok := status.FromError(err); ok {
				if st.Code() != codes.AlreadyExists {
					return fmt.Errorf("error creating profile: %w", err)
				}
				alreadyExists = "Hey, it seems you already tried the quickstart command and created such a profile. " +
					"In case you have registered new repositories, the profile will be already applied to them."
			} else {
				return fmt.Errorf("error creating profile: %w", err)
			}
		}

		// Finish - Confirm profile creation
		cli.PrintCmd(cmd, cli.WarningBanner.Render(stepPromptMsgFinish))

		// Print the "profile already exists" message, if needed
		if alreadyExists != "" {
			cli.PrintCmd(cmd, cli.WarningBanner.Render(alreadyExists))
		} else {
			// Print the profile create result table
			cmd.Println("Profile details (minder profile list -p github):")
			table := profile.InitializeTable(cmd)
			profile.RenderProfileTable(resp.GetProfile(), table)
			table.Render()
		}
		return nil
	},
}

func init() {
	app.RootCmd.AddCommand(cmd)
	cmd.Flags().StringP("project", "r", "", "Project to create the quickstart profile in")
	cmd.Flags().StringP("provider", "p", "github", "Name of the provider")
	cmd.Flags().StringP("token", "t", "", "Personal Access Token (PAT) to use for enrollment")
	cmd.Flags().StringP("owner", "o", "", "Owner to filter on for provider resources")
}
