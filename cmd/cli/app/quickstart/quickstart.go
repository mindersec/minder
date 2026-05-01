// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package quickstart provides the quickstart command for the minder CLI
// which is used to provide the means to quickly get started with minder.
package quickstart

import (
	"context"
	"fmt"
	"os"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mindersec/minder/cmd/cli/app"
	"github.com/mindersec/minder/cmd/cli/app/auth"
	minderprov "github.com/mindersec/minder/cmd/cli/app/provider"
	"github.com/mindersec/minder/cmd/cli/app/repo"
	internalcli "github.com/mindersec/minder/internal/cli"
	ghclient "github.com/mindersec/minder/internal/providers/github/clients"
	"github.com/mindersec/minder/internal/util/cli"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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
	cmd.SetContext(ctx)
	err = repo.RegisterCmd(cmd, []string{})
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

	clonedRepo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL:   repoURL,
		Depth: 1,
	})
	if err != nil {
		return fmt.Errorf("failed to load catalog repo: %w", err)
	}

	worktree, err := clonedRepo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to load catalog repo: %w", err)
	}

	catalog, err := internalcli.LoadCatalogFromFS(worktree.Filesystem, cmd.Printf)
	if err != nil {
		return fmt.Errorf("failed to load catalog: %w", err)
	}

	// Now prompt user AFTER validation
	// Step 3 - Confirm rule type creation
	yes = cli.PrintYesNoPrompt(cmd,
		stepPromptMsgRuleType,
		"Proceed?",
		"Quickstart operation cancelled.",
		true)
	if !yes {
		return nil
	}

	// Step 4 - Confirm profile creation with context
	yes = cli.PrintYesNoPrompt(cmd,
		fmt.Sprintf(stepPromptMsgProfile, strings.Join(registeredRepos, "\n")),
		"Proceed?",
		"Quickstart operation cancelled.",
		true)
	if !yes {
		return nil
	}

	// Apply all resources (transactional flow)
	return applyCatalog(cmd, ruleClient, profileClient, catalog, project)
}

const (
	defaultQuickstartCatalogRepoURL = "https://github.com/mindersec/minder-rules-and-profiles"
)

// applyCatalog creates all rule types and profiles from the catalog via gRPC services.
//
// This function is called AFTER user confirmation and validation. It creates all
// resources in the catalog. If any creation fails, the error is returned immediately.
// The function always prints a summary of created resources, whether new or already
// existing.
func applyCatalog(
	cmd *cobra.Command,
	ruleClient minderv1.RuleTypeServiceClient,
	profileClient minderv1.ProfileServiceClient,
	catalog *internalcli.Catalog,
	project string,
) error {
	ctx, cancel := getQuickstartContext(cmd.Context(), viper.GetViper())
	defer cancel()

	projectContext := &minderv1.Context{
		Project: &project,
	}
	result := applyCatalogResult{
		createdRuleTypes: make([]string, 0, len(catalog.RuleTypes)),
		createdProfiles:  make([]string, 0, len(catalog.Profiles)),
	}

	if err := createCatalogRuleTypes(ctx, cmd, ruleClient, catalog.RuleTypes, projectContext, &result); err != nil {
		rollbackCatalogResources(ctx, ruleClient, profileClient, result)
		return err
	}

	if err := createCatalogProfiles(ctx, cmd, profileClient, catalog.Profiles, projectContext, &result); err != nil {
		rollbackCatalogResources(ctx, ruleClient, profileClient, result)
		return err
	}

	printCatalogSummary(cmd, result)
	return nil
}

type applyCatalogResult struct {
	createdRuleTypes    []string
	createdProfiles     []string
	seenExistingProfile bool
}

func rollbackCatalogResources(
	ctx context.Context,
	ruleClient minderv1.RuleTypeServiceClient,
	profileClient minderv1.ProfileServiceClient,
	result applyCatalogResult,
) {
	for _, profileID := range result.createdProfiles {
		_, _ = profileClient.DeleteProfile(ctx, &minderv1.DeleteProfileRequest{Id: profileID})
	}
	for _, ruleTypeID := range result.createdRuleTypes {
		_, _ = ruleClient.DeleteRuleType(ctx, &minderv1.DeleteRuleTypeRequest{Id: ruleTypeID})
	}
}

func createCatalogRuleTypes(
	ctx context.Context,
	cmd *cobra.Command,
	ruleClient minderv1.RuleTypeServiceClient,
	ruleTypes []*minderv1.RuleType,
	projectContext *minderv1.Context,
	result *applyCatalogResult,
) error {
	for _, ruleType := range ruleTypes {
		ruleType.Context = projectContext

		cmd.Printf("Creating rule type %s...\n", ruleType.GetName())
		resp, err := ruleClient.CreateRuleType(ctx, &minderv1.CreateRuleTypeRequest{RuleType: ruleType})
		if err != nil {
			if st, ok := status.FromError(err); ok && st.Code() == codes.AlreadyExists {
				cmd.Printf("Rule type %s already exists\n", ruleType.GetName())
				continue
			}
			return fmt.Errorf("error creating rule type %s: %w", ruleType.GetName(), err)
		}

		name := resp.GetRuleType().GetName()
		if name == "" {
			name = ruleType.GetName()
		}
		result.createdRuleTypes = append(result.createdRuleTypes, name)
	}

	return nil
}

func createCatalogProfiles(
	ctx context.Context,
	cmd *cobra.Command,
	profileClient minderv1.ProfileServiceClient,
	loadedProfiles []*minderv1.Profile,
	projectContext *minderv1.Context,
	result *applyCatalogResult,
) error {
	for _, profileResource := range loadedProfiles {
		profileResource.Context = projectContext

		cmd.Printf("Creating profile %s...\n", profileResource.GetName())
		resp, err := profileClient.CreateProfile(ctx, &minderv1.CreateProfileRequest{Profile: profileResource})
		if err != nil {
			if st, ok := status.FromError(err); ok && st.Code() == codes.AlreadyExists {
				cmd.Printf("Profile %s already exists\n", profileResource.GetName())
				result.seenExistingProfile = true
				continue
			}
			return fmt.Errorf("error creating profile %s: %w", profileResource.GetName(), err)
		}

		result.createdProfiles = append(result.createdProfiles, resp.GetProfile().GetId())
	}

	return nil
}

func printCatalogSummary(cmd *cobra.Command, result applyCatalogResult) {
	if result.seenExistingProfile {
		cmd.Println(cli.WarningBanner.Render(stepPromptMsgFinishExisting + stepPromptMsgFinishBase))
	} else {
		cmd.Println(cli.WarningBanner.Render(stepPromptMsgFinishOK + stepPromptMsgFinishBase))
	}
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
