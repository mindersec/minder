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
	ghclient "github.com/mindersec/minder/internal/providers/github/clients"
	"github.com/mindersec/minder/internal/util/cli"
	"github.com/mindersec/minder/pkg/api"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	"github.com/mindersec/minder/pkg/fileconvert"
)

const (
	// DefaultPolicyURL points to the OpenSSF security-baseline in minder-rules-and-profiles
	// nolint:lll
	DefaultPolicyURL = "https://github.com/mindersec/minder-rules-and-profiles?profiles/github"

	// nolint:lll
	stepPromptMsgWelcome = `
Welcome! 👋 

You are about to go through the quickstart process for Minder. Throughout this, you will: 

* Authenticate and initialize your account
* Review the baseline security policy to apply
* Enroll your provider (e.g. GitHub)
* Register your repositories
* Apply the security policy and profiles

Let's get started!
`
	// nolint:lll
	stepPromptMsgEnroll = `
Step 2 - Enroll your provider.

This will enroll the provider for your repositories.

Currently Minder works with GitHub, GitLab, and OCI registries, with more providers planned!

The command we are about to run is:

minder provider enroll
`
	// nolint:lll
	stepPromptMsgRegister = `
Step 3 - Register your repositories.

Now that you have enrolled your provider successfully, you can register your repositories with Minder.

The command we are about to run is:

minder repo register
`
	// nolint:lll
	stepPromptMsgApplyPolicy = `
Step 4 - Apply baseline security policy.

We will now apply the security rules and profiles to your registered repositories:

%s

The policy will be applied across your registered repositories.
`
	// nolint:lll
	stepPromptMsgFinishBase = `
You can now continue to explore Minder's features by adding or removing repositories, configuring custom profiles, and reviewing evaluation status.

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
Congratulations! 🎉 You've now successfully configured your baseline security posture in Minder!

`
)

// RuleTypeDetail holds descriptive info about a rule type
type RuleTypeDetail struct {
	Name        string
	DisplayName string
	Description string
}

// ProfileDetail holds descriptive info and remediation setting for a profile
type ProfileDetail struct {
	Name        string
	DisplayName string
	Remediate   string
	Alert       string
}

// PolicySummary describes the resources contained in a policy
type PolicySummary struct {
	RuleTypes []RuleTypeDetail
	Profiles  []ProfileDetail
}

var cmd = &cobra.Command{
	Use:    "quickstart",
	Short:  "Quickstart minder",
	Long:   "The quickstart command provides the means to quickly get started with minder",
	Hidden: true,
	RunE:   cli.GRPCClientWrapRunE(quickstartCommand),
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
	dataSourceClient := minderv1.NewDataSourceServiceClient(conn)

	project := viper.GetString("project")
	provider := viper.GetString("provider")
	policyURL := viper.GetString("policy-url")
	if policyURL == "" {
		policyURL = DefaultPolicyURL
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
		err = loginPromptErrWrapper(cmd, conn, err)
		if err != nil {
			return cli.MessageAndError("", err)
		}
		// User logged in successfully - re-create gRPC connection
		newConn, err := cli.GrpcForCommand(cmd, viper.GetViper())
		if err != nil {
			return err
		}
		defer newConn.Close()

		conn = newConn
		repoClient = minderv1.NewRepositoryServiceClient(conn)
		profileClient = minderv1.NewProfileServiceClient(conn)
		ruleClient = minderv1.NewRuleTypeServiceClient(conn)
		dataSourceClient = minderv1.NewDataSourceServiceClient(conn)
	}

	// Step 1: Fetch and describe policy up-front
	cmd.PrintErrln("Fetching baseline security policy from:", policyURL)
	resources, summary, err := fetchPolicyResources(cmd.Context(), cmd.Printf, policyURL)
	if err != nil {
		return cli.MessageAndError("Error fetching security policy", err)
	}

	describeMsg := describePolicy(policyURL, summary)
	yes = cli.PrintYesNoPrompt(cmd,
		describeMsg,
		"Accept policy and proceed?",
		"Quickstart operation cancelled.",
		true)
	if !yes {
		return nil
	}

	// Step 2: Enroll provider
	yes = cli.PrintYesNoPrompt(cmd,
		stepPromptMsgEnroll,
		"Proceed?",
		"Quickstart operation cancelled.",
		true)
	if !yes {
		return nil
	}

	ctx, cancel := getQuickstartContext(cmd.Context(), viper.GetViper())
	defer cancel()

	ctx = cli.WithRPCClient[minderv1.ProvidersServiceClient](ctx, minderv1.NewProvidersServiceClient(conn))
	ctx = cli.WithRPCClient[minderv1.OAuthServiceClient](ctx, minderv1.NewOAuthServiceClient(conn))
	cmd.SetContext(ctx)
	err = minderprov.EnrollProviderCmd(cmd, []string{})
	if err != nil {
		return cli.MessageAndError("Error enrolling provider", err)
	}

	// Step 3: Register repositories
	yes = cli.PrintYesNoPrompt(cmd,
		stepPromptMsgRegister,
		"Proceed?",
		"Quickstart operation cancelled.",
		true)
	if !yes {
		return nil
	}

	ctx, cancel = getQuickstartContext(cmd.Context(), viper.GetViper())
	defer cancel()

	cmd.SetContext(ctx)
	err = repo.RegisterCmd(cmd, []string{})
	if err != nil {
		return cli.MessageAndError("Error registering repositories", err)
	}

	ctx, cancel = getQuickstartContext(cmd.Context(), viper.GetViper())
	defer cancel()

	// Get the list of all registered repositories
	var registeredRepos []string
	var cursor string
	for range 100 {
		listResp, err := repoClient.ListRepositories(ctx, &minderv1.ListRepositoriesRequest{
			Context: &minderv1.Context{Provider: &provider, Project: &project},
			Cursor:  cursor,
		})
		if err != nil {
			return cli.MessageAndError("Error getting list of repos", err)
		}
		for _, result := range listResp.Results {
			r := fmt.Sprintf("%s/%s", result.Owner, result.Name)
			registeredRepos = append(registeredRepos, r)
		}
		cursor = listResp.Cursor
		if cursor == "" {
			break
		}
	}

	// Step 4: Apply policy resources to registered repositories
	return applyPolicyResources(cmd, ruleClient, profileClient, dataSourceClient, project, registeredRepos, resources)
}

// fetchPolicyResources retrieves resources from a remote git URL, HTTP URL, or local path
func fetchPolicyResources(
	ctx context.Context,
	printer fileconvert.Printer,
	rawURL string,
) ([]minderv1.ResourceMeta, PolicySummary, error) {
	var resources []minderv1.ResourceMeta
	var err error

	// If git URL style, clone and extract
	if _, _, parseErr := fileconvert.ParseGitURL(rawURL); parseErr == nil {
		resources, err = fileconvert.ResourcesFromGit(ctx, printer, rawURL)
	} else {
		resources, err = fileconvert.ResourcesFromPaths(ctx, printer, rawURL)
	}

	if err != nil {
		return nil, PolicySummary{}, err
	}
	if len(resources) == 0 {
		return nil, PolicySummary{}, fmt.Errorf("no resources found at %s", rawURL)
	}

	summary := summarizeResources(resources)
	return resources, summary, nil
}

// summarizeResources extracts rule types and profiles for human review
func summarizeResources(resources []minderv1.ResourceMeta) PolicySummary {
	var summary PolicySummary
	for _, res := range resources {
		switch r := res.(type) {
		case *minderv1.RuleType:
			detail := RuleTypeDetail{
				Name:        r.GetName(),
				DisplayName: r.GetDisplayName(),
				Description: r.GetDescription(),
			}
			summary.RuleTypes = append(summary.RuleTypes, detail)
		case *minderv1.Profile:
			detail := ProfileDetail{
				Name:        r.GetName(),
				DisplayName: r.GetDisplayName(),
				Remediate:   r.GetRemediate(),
				Alert:       r.GetAlert(),
			}
			if detail.Remediate == "" {
				detail.Remediate = "off"
			}
			if detail.Alert == "" {
				detail.Alert = "on"
			}
			summary.Profiles = append(summary.Profiles, detail)
		}
	}
	return summary
}

// describePolicy formats rule types and profiles cleanly with markdown formatting
func describePolicy(policyURL string, s PolicySummary) string {
	var b strings.Builder
	b.WriteString("\nStep 1 - Review Security Baseline Policy.\n\n")
	fmt.Fprintf(&b, "Source: %s\n\n", policyURL)
	b.WriteString("The following baseline controls will be configured in your project:\n")

	if len(s.RuleTypes) > 0 {
		b.WriteString("\nRule Types:\n")
		for _, rt := range s.RuleTypes {
			displayName := rt.DisplayName
			if displayName == "" {
				displayName = rt.Name
			}
			if rt.Description != "" {
				fmt.Fprintf(&b, "  • %s: %s\n", displayName, rt.Description)
			} else {
				fmt.Fprintf(&b, "  • %s\n", displayName)
			}
		}
	}

	if len(s.Profiles) > 0 {
		b.WriteString("\nProfiles:\n")
		for _, p := range s.Profiles {
			displayName := p.DisplayName
			if displayName == "" {
				displayName = p.Name
			}
			fmt.Fprintf(&b, "  • %s (Remediate: %s, Alert: %s)\n", displayName, p.Remediate, p.Alert)
		}
	}

	b.WriteString("\nThese rules continuously evaluate your repositories and alert or remediate non-compliant settings.\n")
	return b.String()
}

// applyPolicyResources re-uses Upsert helpers from api package to apply resources
//
//nolint:gocyclo
func applyPolicyResources(
	cmd *cobra.Command,
	ruleClient minderv1.RuleTypeServiceClient,
	profileClient minderv1.ProfileServiceClient,
	dataSourceClient minderv1.DataSourceServiceClient,
	project string,
	registeredRepos []string,
	resources []minderv1.ResourceMeta,
) error {
	reposList := strings.Join(registeredRepos, "\n")
	if len(registeredRepos) == 0 {
		reposList = "(No repositories registered yet)"
	}

	yes := cli.PrintYesNoPrompt(cmd,
		fmt.Sprintf(stepPromptMsgApplyPolicy, reposList),
		"Proceed?",
		"Quickstart operation cancelled.",
		true)
	if !yes {
		return nil
	}

	cmd.PrintErrln("Applying baseline security policy...")

	v1Context := &minderv1.Context{
		Project: &project,
	}
	v2Context := &minderv1.ContextV2{
		ProjectId: project,
	}

	var dataSources []*minderv1.DataSource
	var ruleTypes []*minderv1.RuleType
	var profiles []*minderv1.Profile

	for _, res := range resources {
		switch r := res.(type) {
		case *minderv1.DataSource:
			r.Context = v2Context
			dataSources = append(dataSources, r)
		case *minderv1.RuleType:
			r.Context = v1Context
			ruleTypes = append(ruleTypes, r)
		case *minderv1.Profile:
			r.Context = v1Context
			profiles = append(profiles, r)
		}
	}

	// 1. DataSources
	for _, ds := range dataSources {
		ctx, cancel := getQuickstartContext(cmd.Context(), viper.GetViper())
		err := api.UpsertDataSource(ctx, dataSourceClient, ds)
		cancel()
		if err != nil {
			return cli.MessageAndError(fmt.Sprintf("Unable to apply data source %s", ds.GetName()), err)
		}
		cmd.PrintErrf("Applied data source: %s\n", ds.GetName())
	}

	// 2. RuleTypes
	for _, rt := range ruleTypes {
		ctx, cancel := getQuickstartContext(cmd.Context(), viper.GetViper())
		err := api.UpsertRuleType(ctx, ruleClient, rt)
		cancel()
		if err != nil {
			return cli.MessageAndError(fmt.Sprintf("Unable to apply rule type %s", rt.GetName()), err)
		}
		cmd.PrintErrf("Applied rule type: %s\n", rt.GetName())
	}

	// 3. Profiles
	for _, p := range profiles {
		ctx, cancel := getQuickstartContext(cmd.Context(), viper.GetViper())
		err := api.UpsertProfile(ctx, profileClient, p)
		cancel()
		if err != nil {
			return cli.MessageAndError(fmt.Sprintf("Unable to apply profile %s", p.GetName()), err)
		}
		cmd.PrintErrf("Applied profile: %s\n", p.GetName())
	}

	// Finish
	cmd.PrintErrln(cli.SuccessBanner.Render(stepPromptMsgFinishOK + stepPromptMsgFinishBase))
	for _, prof := range profiles {
		cmd.PrintErrf("Profile: %s\n", prof.GetName())
		table := profile.NewProfileRulesTable(cmd.OutOrStdout())
		profile.RenderProfileRulesTable(prof, table)
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
	cmd.Flags().String("policy-url", DefaultPolicyURL, "URL or path to fetch baseline policy from")

	// Bind flags
	if err := viper.BindPFlag("token", cmd.Flags().Lookup("token")); err != nil {
		cmd.PrintErrf("error: %s", err)
		os.Exit(1)
	}
	if err := viper.BindPFlag("owner", cmd.Flags().Lookup("owner")); err != nil {
		cmd.PrintErrf("error: %s", err)
		os.Exit(1)
	}
	if err := viper.BindPFlag("policy-url", cmd.Flags().Lookup("policy-url")); err != nil {
		cmd.PrintErrf("error: %s", err)
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
			cmnd.PrintErrln(cli.SuccessBanner.Render("You have been successfully registered. Welcome!"))
			return nil
		}
	}
	// Not a grpc status error, return the original error
	return inErr
}
