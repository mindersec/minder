// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/cmd/cli/app/auth"
	"github.com/mindersec/minder/cmd/cli/app/profile"
	minderprov "github.com/mindersec/minder/cmd/cli/app/provider"
	"github.com/mindersec/minder/cmd/cli/app/repo"
	internalrepo "github.com/mindersec/minder/cmd/cli/internal/repo"
	ghclient "github.com/mindersec/minder/internal/providers/github/clients"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/profiles"
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
* https://github.com/mindersec/minder
Official documentation:
* https://mindersec.github.io
CLI commands:
* https://mindersec.github.io/ref/cli/minder
Minder Rules & profiles:
* https://github.com/mindersec/minder-rules-and-profiles

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
func quickstartCommand(
	_ context.Context,
	cmd *cobra.Command,
	_ []string,
	conn *grpc.ClientConn,
) error {
	var err error
	repoClient := minderv1.NewRepositoryServiceClient(conn)
	profileClient := minderv1.NewProfileServiceClient(conn)
	ruleClient := minderv1.NewRuleTypeServiceClient(conn)

	project := viper.GetString("project")
	provider := viper.GetString("provider")

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
		err = loginPromptErrWrapper(cmd, conn, err)
		if err != nil {
			return cli.MessageAndError("", err)
		}
		// User logged in successfully
		// We now have to re-create the gRPC connection
		newConn, err := cli.GrpcForCommand(cmd, viper.GetViper())
		if err != nil {
			return err
		}
		defer newConn.Close()

		// Update the existing clients with the new connection
		conn = newConn
		repoClient = minderv1.NewRepositoryServiceClient(conn)
		profileClient = minderv1.NewProfileServiceClient(conn)
		ruleClient = minderv1.NewRuleTypeServiceClient(conn)
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
	err = minderprov.EnrollProviderCommand(ctx, cmd, []string{}, conn)
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
	err = repo.RegisterCmd(ctx, cmd, []string{}, conn)
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
	repoURL, err := cmd.Flags().GetString("catalog-repo")
	if err != nil {
		return err
	}
	if repoURL == "" {
		repoURL = defaultQuickstartCatalogRepoURL
	}

	return loadCatalog(cmd, ruleClient, profileClient, provider, project, registeredRepos, repoURL)
}

const (
	quickstartRuleTypeFilePath      = "rule-types/github/secret_scanning.yaml"
	quickstartProfileFilePath       = "profiles/github/profile.yaml"
	defaultQuickstartCatalogRepoURL = "https://github.com/mindersec/minder-rules-and-profiles"
)

// loadCatalog drives the catalog portion of the quickstart flow.
//
// It first asks the user to confirm creation of the initial quickstart
// resources (the secret_scanning rule type and its profile), then attempts
// to source those resources from the configured catalog repository using
// loadCatalogFromRepo.
//
// If cloning the catalog repository or reading/parsing its files fails for
// any reason, loadCatalog prints a warning and transparently falls back to
// runExistingFlow, which uses the embedded quickstart YAML files shipped
// with the CLI. This preserves the original quickstart behavior while
// preferring the up-to-date remote catalog when available.
func loadCatalog(
	cmd *cobra.Command,
	ruleClient minderv1.RuleTypeServiceClient,
	profileClient minderv1.ProfileServiceClient,
	provider string,
	project string,
	registeredRepos []string,
	repoURL string,
) error {
	// Step 3 - Confirm rule type creation
	yes := cli.PrintYesNoPrompt(cmd,
		stepPromptMsgRuleType,
		"Proceed?",
		"Quickstart operation cancelled.",
		true)
	if !yes {
		return nil
	}

	if err := loadCatalogFromRepo(cmd, ruleClient, profileClient, provider, project, registeredRepos, repoURL); err != nil {
		cmd.Printf("Warning: failed to load quickstart catalog from %s: %v\n", repoURL, err)
		cmd.Printf("Falling back to embedded quickstart catalog.\n")
		return runExistingFlow(cmd, ruleClient, profileClient, provider, project, registeredRepos)
	}

	return nil
}

// loadCatalogFromRepo loads the quickstart rule type and profile directly
// from the configured Git repository.
//
// The function clones the catalog repository in memory, opens the
// quickstart rule type and profile YAML files from the in-memory
// filesystem, parses them into protobuf/CLI profile structures, applies
// the current provider/project context, and then creates the resources via
// the RuleType and Profile gRPC services.
//
// It mirrors the prompts and output of the original quickstart flow while
// sourcing the definitions from a remote catalog instead of the embedded
// YAML. Any error is propagated to the caller so that a higher-level
// fallback (to the embedded flow) can be applied.
func loadCatalogFromRepo(
	cmd *cobra.Command,
	ruleClient minderv1.RuleTypeServiceClient,
	profileClient minderv1.ProfileServiceClient,
	provider string,
	project string,
	registeredRepos []string,
	repoURL string,
) error {
	catalogRepo, err := internalrepo.CloneInMemory(repoURL)
	if err != nil {
		return fmt.Errorf("failed to clone catalog repository: %w", err)
	}

	worktree, err := catalogRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	fs := worktree.Filesystem

	rtReader, err := fs.Open(quickstartRuleTypeFilePath)
	if err != nil {
		return fmt.Errorf("failed to open rule type file: %w", err)
	}
	defer rtReader.Close()

	rt := &minderv1.RuleType{}
	if err := minderv1.ParseResource(rtReader, rt); err != nil {
		return fmt.Errorf("failed to parse rule type: %w", err)
	}

	rt.Context = &minderv1.Context{
		Provider: &provider,
		Project:  &project,
	}

	ctx, cancel := getQuickstartContext(cmd.Context(), viper.GetViper())
	defer cancel()

	cmd.Printf("Creating rule type from remote catalog...\n")
	_, err = ruleClient.CreateRuleType(ctx, &minderv1.CreateRuleTypeRequest{RuleType: rt})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			if st.Code() != codes.AlreadyExists {
				return fmt.Errorf("error creating rule type from remote catalog: %w", err)
			}
			cmd.Println("Rule type secret_scanning already exists")
		} else {
			return cli.MessageAndError("error creating rule type", err)
		}
	}

	yes := cli.PrintYesNoPrompt(cmd,
		fmt.Sprintf(stepPromptMsgProfile, strings.Join(registeredRepos, "\n")),
		"Proceed?",
		"Quickstart operation cancelled.",
		true)
	if !yes {
		return nil
	}

	cmd.Printf("Creating profile from remote catalog...\n")
	profileReader, err := fs.Open(quickstartProfileFilePath)
	if err != nil {
		return fmt.Errorf("failed to open profile file: %w", err)
	}
	defer profileReader.Close()

	p, err := profiles.ParseYAML(profileReader)
	if err != nil {
		return fmt.Errorf("failed to parse profile: %w", err)
	}

	p.Context = &minderv1.Context{
		Provider: &provider,
		Project:  &project,
	}

	ctx, cancel = getQuickstartContext(cmd.Context(), viper.GetViper())
	defer cancel()

	alreadyExists := false
	resp, err := profileClient.CreateProfile(ctx, &minderv1.CreateProfileRequest{Profile: p})
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

	if alreadyExists {
		cmd.Println(cli.WarningBanner.Render(stepPromptMsgFinishExisting + stepPromptMsgFinishBase))
	} else {
		cmd.Println(cli.WarningBanner.Render(stepPromptMsgFinishOK + stepPromptMsgFinishBase))
		cmd.Println("Profile details (minder profile list):")
		table := profile.NewProfileRulesTable(cmd.OutOrStdout())
		profile.RenderProfileRulesTable(resp.GetProfile(), table)
		table.Render()
	}

	return nil
}

// runExistingFlow contains the original quickstart catalog logic that uses
// the embedded secret_scanning rule type and profile YAML files.
//
// This function is used as a safe fallback when loading the catalog from
// the configured repository fails. It recreates the previous Step 3 and
// Step 4 behavior by:
//   - reading secret_scanning.yaml and profile.yaml from the embedded FS,
//   - parsing them into the appropriate rule type and profile structures,
//   - applying the current provider/project context, and
//   - creating the resources via the corresponding gRPC services, including
//     handling AlreadyExists responses and printing the final banners and
//     profile details table.
func runExistingFlow(
	cmd *cobra.Command,
	ruleClient minderv1.RuleTypeServiceClient,
	profileClient minderv1.ProfileServiceClient,
	provider string,
	project string,
	registeredRepos []string,
) error {
	cmd.Println("Creating rule type...")
	reader, err := content.Open("embed/secret_scanning.yaml")
	if err != nil {
		return cli.MessageAndError("error opening rule type", err)
	}

	rt := &minderv1.RuleType{}
	if err := minderv1.ParseResource(reader, rt); err != nil {
		return cli.MessageAndError("error parsing rule type", err)
	}

	if rt.Context == nil {
		rt.Context = &minderv1.Context{}
	}

	rt.Context = &minderv1.Context{
		Provider: &provider,
		Project:  &project,
	}

	ctx, cancel := getQuickstartContext(cmd.Context(), viper.GetViper())
	defer cancel()

	_, err = ruleClient.CreateRuleType(ctx, &minderv1.CreateRuleTypeRequest{RuleType: rt})
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

	yes := cli.PrintYesNoPrompt(cmd,
		fmt.Sprintf(stepPromptMsgProfile, strings.Join(registeredRepos, "\n")),
		"Proceed?",
		"Quickstart operation cancelled.",
		true)
	if !yes {
		return nil
	}

	cmd.Println("Creating profile...")
	reader, err = content.Open("embed/profile.yaml")
	if err != nil {
		return cli.MessageAndError("error opening profile", err)
	}

	p, err := profiles.ParseYAML(reader)
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

	ctx, cancel = getQuickstartContext(cmd.Context(), viper.GetViper())
	defer cancel()

	alreadyExists := false
	resp, err := profileClient.CreateProfile(ctx, &minderv1.CreateProfileRequest{Profile: p})
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

	if alreadyExists {
		cmd.Println(cli.WarningBanner.Render(stepPromptMsgFinishExisting + stepPromptMsgFinishBase))
	} else {
		cmd.Println(cli.WarningBanner.Render(stepPromptMsgFinishOK + stepPromptMsgFinishBase))
		cmd.Println("Profile details (minder profile list):")
		table := profile.NewProfileRulesTable(cmd.OutOrStdout())
		profile.RenderProfileRulesTable(resp.GetProfile(), table)
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
	cmd.Flags().String("catalog-repo", "", "Repository URL to load quickstart catalog from")
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

func loginPromptErrWrapper(
	cmnd *cobra.Command,
	conn *grpc.ClientConn,
	inErr error,
) error {
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
		if rpcStatus.Code() == codes.NotFound {
			// User is authenticated but not yet registered; auto-register so quickstart can proceed.
			userClient := minderv1.NewUserServiceClient(conn)
			quickstartCtx, cancel := getQuickstartContext(cmnd.Context(), viper.GetViper())
			defer cancel()
			if _, err := userClient.CreateUser(quickstartCtx, &minderv1.CreateUserRequest{}); err != nil {
				return cli.MessageAndError("Error registering user", err)
			}
			cmnd.Println(cli.SuccessBanner.Render("You have been successfully registered. Welcome!"))
			return nil
		}
	}
	// Not a grpc status error, return the original error
	return inErr
}
