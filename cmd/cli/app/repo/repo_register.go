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

package repo

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/cli/table"
	"github.com/stacklok/minder/internal/util/cli/table/layouts"
	"github.com/stacklok/minder/internal/util/ptr"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var repoRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a repository",
	Long:  `The repo register subcommand is used to register a repo within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(RegisterCmd),
}

// RegisterCmd represents the register command to register a repo with minder
//
//nolint:gocyclo
func RegisterCmd(ctx context.Context, cmd *cobra.Command, _ []string, conn *grpc.ClientConn) error {
	client := minderv1.NewRepositoryServiceClient(conn)

	provider := viper.GetString("provider")
	project := viper.GetString("project")
	inputRepoList := viper.GetStringSlice("name")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	for _, repo := range inputRepoList {
		if err := cli.ValidateRepositoryName(repo); err != nil {
			return cli.MessageAndError("Invalid repository name", err)
		}
	}

	// Fetch remote repos, both registered and unregistered.
	repos, err := fetchRepos(ctx, provider, project, client)
	if err != nil {
		return cli.MessageAndError("Error getting registered repos", err)
	}

	// Maps for filtering
	registeredRepos := make(map[string]*minderv1.UpstreamRepositoryRef)
	unregisteredRepos := make(map[string]*minderv1.UpstreamRepositoryRef)
	for _, repo := range repos {
		key := cli.GetRepositoryName(repo.Owner, repo.Name)
		if repo.Registered {
			registeredRepos[key] = repo
		} else {
			unregisteredRepos[key] = repo
		}
	}

	// No repos left to register, exit cleanly
	if len(unregisteredRepos) == 0 {
		cmd.Println("No repos left to register")
		return nil
	}

	var selectedRepos []*minderv1.UpstreamRepositoryRef
	if len(inputRepoList) > 0 {
		// Repositories are provided as --name options
		for _, repo := range inputRepoList {
			// Repo was already registered, report it to
			// user and move on
			if registeredRepos[repo] != nil {
				cmd.Printf("Repository %s is already registered\n", repo)
			}

			// Repo was not already registered, add it to
			// those to process.
			if repoRef := unregisteredRepos[repo]; repoRef != nil {
				selectedRepos = append(selectedRepos, repoRef)
			}
		}
	} else {
		cmd.Printf(
			"Found %d remote repositories: %d registered and %d unregistered.\n",
			len(registeredRepos)+len(unregisteredRepos),
			len(registeredRepos),
			len(unregisteredRepos),
		)

		var err error
		selectedRepos, err = selectReposInteractively(
			cmd,
			unregisteredRepos,
		)
		if err != nil {
			return cli.MessageAndError("Error getting selected repositories", err)
		}
	}

	results, warnings := registerRepos(project, client, selectedRepos)
	printWarnings(cmd, warnings)

	printRepoRegistrationStatus(cmd, results)
	return nil
}

func fetchRepos(
	ctx context.Context,
	provider string,
	project string,
	client minderv1.RepositoryServiceClient,
) ([]*minderv1.UpstreamRepositoryRef, error) {
	var provPtr *string
	if provider != "" {
		provPtr = &provider
	}

	resp, err := client.ListRemoteRepositoriesFromProvider(
		ctx,
		&minderv1.ListRemoteRepositoriesFromProviderRequest{
			Context: &minderv1.Context{
				Provider: provPtr,
				Project:  &project,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return resp.Results, nil
}

func selectReposInteractively(
	cmd *cobra.Command,
	unregisteredRepos map[string]*minderv1.UpstreamRepositoryRef,
) ([]*minderv1.UpstreamRepositoryRef, error) {
	effectiveRepos := make([]*minderv1.UpstreamRepositoryRef, 0)

	repoNames := make([]string, 0, len(unregisteredRepos))
	for repoName := range unregisteredRepos {
		repoNames = append(repoNames, repoName)
	}

	selected, err := cli.MultiSelect(repoNames)
	if err != nil {
		return nil, err
	}

	for _, name := range selected {
		effectiveRepos = append(effectiveRepos, unregisteredRepos[name])
	}

	if len(effectiveRepos) == 0 {
		cmd.Println("No repositories selected")
	}

	return effectiveRepos, nil
}

func registerRepos(
	project string,
	client minderv1.RepositoryServiceClient,
	repos []*minderv1.UpstreamRepositoryRef,
) ([]*minderv1.RegisterRepoResult, []string) {
	var results []*minderv1.RegisterRepoResult
	var warnings []string
	for _, repo := range repos {
		result, err := client.RegisterRepository(
			context.Background(),
			&minderv1.RegisterRepositoryRequest{
				Context: &minderv1.Context{
					Provider: ptr.Ptr(repo.GetContext().GetProvider()),
					Project:  &project,
				},
				Repository: repo,
			},
		)

		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Error registering repository %s: %s", repo.Name, err))
			continue
		}
		results = append(results, result.Result)
	}

	return results, warnings
}

func printRepoRegistrationStatus(cmd *cobra.Command, results []*minderv1.RegisterRepoResult) {
	// If there were no results, print a message and return
	if len(results) == 0 {
		cmd.Println("No repositories registered")
		return
	}

	t := table.New(table.Simple, layouts.Default, []string{"Repository", "Status", "Message"})
	for _, result := range results {
		// in the case of a malformed response, skip over it to avoid segfaulting
		if result.Repository == nil {
			cmd.Printf("Skipping malformed response: %v", result)
		}
		row := []string{cli.GetRepositoryName(result.Repository.Owner, result.Repository.Name)}
		if result.Status.Success {
			row = append(row, "Registered")
		} else {
			row = append(row, "Failed")
		}

		if result.Status.Error != nil {
			row = append(row, *result.Status.Error)
		} else {
			row = append(row, "")
		}
		t.AddRow(row...)
	}
	t.Render()
}

func printWarnings(cmd *cobra.Command, warnings []string) {
	for _, warning := range warnings {
		cmd.Println(warning)
	}
}

func init() {
	RepoCmd.AddCommand(repoRegisterCmd)
	// Flags
	repoRegisterCmd.Flags().StringSliceP("name", "n", []string{}, "List of repository names to register, i.e owner/repo,owner/repo")
}
