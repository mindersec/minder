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
	"context"
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/cmd/cli/app/auth"
	"github.com/stacklok/minder/cmd/cli/app/profile"
	minderprov "github.com/stacklok/minder/cmd/cli/app/provider"
	"github.com/stacklok/minder/cmd/cli/app/repo"
	"github.com/stacklok/minder/internal/engine"
	ghclient "github.com/stacklok/minder/internal/providers/github"
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

minder provider enroll
`
	// nolint:lll
	stepPromptMsgRegister = `
Step 2 - Register your repositories.

Now that you have enrolled your provider successfully, you can register your repositories.

The command we are about to do is the following:

minder repo register
`
	// nolint:lll
	stepPromptMsgRuleType = `
Step 3 - Create your first rule type - secret_scanning.

Now that you have registered your repositories with Minder, let's create your first rule type!

For the purpose of this quickstart, we are going to use a rule of type "secret_scanning" (secret_scanning.yaml).
Secret scanning is about protecting you from accidentally leaking secrets in your repository.

The command we are about to do is the following:

minder ruletype create -f secret_scanning.yaml
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
	stepPromptMsgFinishBase = `
You can now continue to explore Minder's features by adding or removing more repositories, create custom profiles with various rules, and much more.

For more information about Minder, see the following resources:

GitHub:
* https://github.com/stacklok/minder
Official documentation:
* https://minder-docs.stacklok.dev
CLI commands:
* https://minder-docs.stacklok.dev/ref/cli/minder
Minder Rules & profiles:
* https://github.com/stacklok/minder-rules-and-profiles

Thank you for using Minder!
`
	// nolint:lll
	stepPromptMsgFinishOK = `
Congratulations! 🎉 You've now successfully created your first profile in Minder!

`
	// nolint:lll
	stepPromptMsgFinishExisting = `
Congratulations! 🎉 It seems you already tried the quickstart command and created such a profile, so we skipped the profile creation step.

In case you have registered new repositories during this flow, the profile will be applied to them too.

`
)

//go:embed embed*
var content embed.FS

var cmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Quickstart minder",
	Long:  "The quickstart command provide the means to quickly get started with minder",
	RunE:  cli.GRPCClientWrapRunE(quickstartCommand),
}

// quickstartCommand is the quickstart command
//
//nolint:gocyclo
func quickstartCommand(_ context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	var err error
	repoClient := minderv1.NewRepositoryServiceClient(conn)
	profileClient := minderv1.NewProfileServiceClient(conn)

	project := viper.GetString("project")
	provider := viper.GetString("provider")

	// Ensure provider is supported
	if !app.IsProviderSupported(provider) {
		return cli.MessageAndError(fmt.Sprintf("Provider %s is not supported yet", provider), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// Confirm user wants to go through the quickstart process
	yes := cli.PrintYesNoPrompt(cmd,
		stepPromptMsgWelcome,
		"Proceed?",
		"Quickstart operation cancelled.",
		true)
	if !yes {
		return nil
	}

	// Ensure user is logged in
	userClient := minderv1.NewUserServiceClient(conn)
	_, err = userClient.GetUser(cmd.Context(), &minderv1.GetUserRequest{})
	if err != nil {
		err = loginPromptErrWrapper(cmd, err)
		if err != nil {
			return cli.MessageAndError("", err)
		}
		// User logged in successfully
		// We now have to re-create the gRPC connection
		newConn, err := cli.GrpcForCommand(viper.GetViper())
		if err != nil {
			return err
		}
		defer newConn.Close()

		// Update the existing clients with the new connection
		conn = newConn
		repoClient = minderv1.NewRepositoryServiceClient(conn)
		profileClient = minderv1.NewProfileServiceClient(conn)
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

	// New context so we don't time out between steps
	ctx, cancel := getQuickstartContext(cmd.Context(), viper.GetViper())
	defer cancel()

	// Enroll provider
	err = minderprov.EnrollProviderCommand(ctx, cmd, conn)
	if err != nil {
		return cli.MessageAndError("Error enrolling provider", err)
	}

	// Step 2 - Confirm repository registration
	yes = cli.PrintYesNoPrompt(cmd,
		stepPromptMsgRegister,
		"Proceed?",
		"Quickstart operation cancelled.",
		true)
	if !yes {
		return nil
	}

	// New context so we don't time out between steps
	ctx, cancel = getQuickstartContext(cmd.Context(), viper.GetViper())
	defer cancel()

	// Prompt to register repositories
	err = repo.RegisterCmd(ctx, cmd, conn)
	if err != nil {
		return cli.MessageAndError("Error registering repositories", err)
	}

	// New context so we don't time out between steps
	ctx, cancel = getQuickstartContext(cmd.Context(), viper.GetViper())
	defer cancel()

	// Get the list of all registered repositories
	listResp, err := repoClient.ListRepositories(ctx, &minderv1.ListRepositoriesRequest{
		Context: &minderv1.Context{Provider: &provider, Project: &project},
	})
	if err != nil {
		return cli.MessageAndError("Error getting list of repos", err)
	}

	var registeredRepos []string
	for _, result := range listResp.Results {
		r := fmt.Sprintf("%s/%s", result.Owner, result.Name)
		registeredRepos = append(registeredRepos, r)
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

	// Creating the rule type
	cmd.Println("Creating rule type...")

	// Load the rule type from the embedded file system
	reader, err := content.Open("embed/secret_scanning.yaml")
	if err != nil {
		return cli.MessageAndError("error opening rule type", err)
	}

	rt, err := minderv1.ParseRuleType(reader)
	if err != nil {
		return cli.MessageAndError("error parsing rule type", err)
	}

	if rt.Context == nil {
		rt.Context = &minderv1.Context{}
	}

	rt.Context = &minderv1.Context{
		Provider: &provider,
		Project:  &project,
	}

	// New context so we don't time out between steps
	ctx, cancel = getQuickstartContext(cmd.Context(), viper.GetViper())
	defer cancel()

	// Create the rule type in minder
	_, err = profileClient.CreateRuleType(ctx, &minderv1.CreateRuleTypeRequest{
		RuleType: rt,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			if st.Code() != codes.AlreadyExists {
				return fmt.Errorf("error creating rule type from: %w", err)
			}
			cmd.Println("Rule type secret_scanning already exists")
		} else {
			return cli.MessageAndError("error creating rule type", err)
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
	reader, err = content.Open("embed/profile.yaml")
	if err != nil {
		return cli.MessageAndError("error opening profile", err)
	}

	// Load the profile from the embedded file system
	p, err := engine.ParseYAML(reader)
	if err != nil {
		return cli.MessageAndError("error parsing profile", err)
	}

	if p.Context == nil {
		p.Context = &minderv1.Context{}
	}

	p.Context = &minderv1.Context{
		Provider: &provider,
		Project:  &project,
	}

	// New context so we don't time out between steps
	ctx, cancel = getQuickstartContext(cmd.Context(), viper.GetViper())
	defer cancel()

	alreadyExists := false
	// Create the profile in minder
	resp, err := profileClient.CreateProfile(ctx, &minderv1.CreateProfileRequest{
		Profile: p,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			if st.Code() != codes.AlreadyExists {
				return cli.MessageAndError("error creating profile", err)
			}
			alreadyExists = true
		} else {
			return cli.MessageAndError("error creating profile", err)
		}
	}

	// Finish - Confirm profile creation
	if alreadyExists {
		// Print the "profile already exists" message
		cmd.Println(cli.WarningBanner.Render(stepPromptMsgFinishExisting + stepPromptMsgFinishBase))
	} else {
		// Print the "profile created" message
		cmd.Println(cli.WarningBanner.Render(stepPromptMsgFinishOK + stepPromptMsgFinishBase))
		// Print the profile create result table
		cmd.Println("Profile details (minder profile list):")
		table := profile.NewProfileTable()
		profile.RenderProfileTable(resp.GetProfile(), table)
		table.Render()
	}
	return nil
}

func init() {
	app.RootCmd.AddCommand(cmd)
	// Flags
	cmd.Flags().StringP("provider", "p", ghclient.Github, "Name of the provider, i.e. github")
	cmd.Flags().StringP("project", "j", "", "ID of the project")
	cmd.Flags().StringP("token", "t", "", "Personal Access Token (PAT) to use for enrollment")
	cmd.Flags().StringP("owner", "o", "", "Owner to filter on for provider resources")
	// Bind flags
	if err := viper.BindPFlag("token", cmd.Flags().Lookup("token")); err != nil {
		cmd.Printf("error: %s", err)
		os.Exit(1)
	}
	if err := viper.BindPFlag("owner", cmd.Flags().Lookup("owner")); err != nil {
		cmd.Printf("error: %s", err)
		os.Exit(1)
	}
}

func getQuickstartContext(ctx context.Context, v *viper.Viper) (context.Context, context.CancelFunc) {
	return cli.GetAppContextWithTimeoutDuration(ctx, v, 30)
}

func loginPromptErrWrapper(cmnd *cobra.Command, inErr error) error {
	// Check if the error is unauthenticated, if so, prompt the user to log in
	if rpcStatus, ok := status.FromError(inErr); ok {
		if rpcStatus.Code() == codes.Unauthenticated {
			// Prompt to log in
			yes := cli.PrintYesNoPrompt(cmnd,
				"It seems you are logged out. Would you like to log in now?",
				"Proceed?",
				"Quickstart operation cancelled.",
				true)
			if yes {
				// Run the login command
				err := auth.LoginCommand(cmnd, []string{})
				if err != nil {
					return err
				}
				// Logged in successfully, return nil, so we can continue forward
				return nil
			}
			// User chose not to log in, return the original error
			return inErr
		}
	}
	// Not a grpc status error, return the original error
	return inErr
}
