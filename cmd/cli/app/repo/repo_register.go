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
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/sets"

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

	alreadyRegisteredRepos, err := fetchAlreadyRegisteredRepos(ctx, provider, project, client)
	if err != nil {
		return cli.MessageAndError("Error getting list of registered repos", err)
	}

	unregisteredInputRepos, warnings := getUnregisteredInputRepos(inputRepoList, alreadyRegisteredRepos)
	printWarnings(cmd, warnings)

	// All input repos are already registered
	if len(inputRepoList) > 0 && len(unregisteredInputRepos) == 0 {
		return nil
	}

	var selectedRepos []*minderv1.UpstreamRepositoryRef
	if len(unregisteredInputRepos) > 0 {
		for _, repo := range unregisteredInputRepos {
			owner, name := cli.GetNameAndOwnerFromRepository(repo)
			selectedRepos = append(selectedRepos, &minderv1.UpstreamRepositoryRef{
				Owner: owner,
				Name:  name,
			})
		}
	} else {
		var err error
		selectedRepos, err = getSelectedReposToRegister(
			ctx, cmd, provider, project, client, alreadyRegisteredRepos, unregisteredInputRepos)
		if err != nil {
			return cli.MessageAndError("Error getting selected repositories", err)
		}
	}

	results, warnings := registerSelectedRepos(project, client, selectedRepos)
	printWarnings(cmd, warnings)

	printRepoRegistrationStatus(cmd, results)
	return nil
}

func getSelectedReposToRegister(
	ctx context.Context, cmd *cobra.Command, provider, project string, client minderv1.RepositoryServiceClient,
	alreadyRegisteredRepos sets.Set[string], unregisteredInputRepos []string) ([]*minderv1.UpstreamRepositoryRef, error) {
	remoteRepositories, err := fetchRemoteRepositoriesFromProvider(ctx, provider, project, client)
	if err != nil {
		return nil, cli.MessageAndError("Error getting list of remote repos", err)
	}

	unregisteredRemoteRepositories := getUnregisteredRemoteRepositories(remoteRepositories, alreadyRegisteredRepos)

	cmd.Printf("Found %d remote repositories: %d registered and %d unregistered.\n",
		len(remoteRepositories), len(alreadyRegisteredRepos), len(unregisteredRemoteRepositories))

	selectedRepos, warnings, err := getSelectedRepositories(unregisteredRemoteRepositories, unregisteredInputRepos)
	if err != nil {
		return nil, cli.MessageAndError("Error getting selected repositories", err)
	}
	printWarnings(cmd, warnings)

	return selectedRepos, nil
}

func fetchAlreadyRegisteredRepos(ctx context.Context, provider, project string, client minderv1.RepositoryServiceClient) (
	sets.Set[string], error) {
	alreadyRegisteredRepos, err := client.ListRepositories(ctx, &minderv1.ListRepositoriesRequest{
		Context: &minderv1.Context{Provider: &provider, Project: &project},
	})
	if err != nil {
		return nil, err
	}

	alreadyRegisteredReposSet := sets.New[string]()
	for _, repo := range alreadyRegisteredRepos.Results {
		alreadyRegisteredReposSet.Insert(cli.GetRepositoryName(repo.Owner, repo.Name))
	}

	return alreadyRegisteredReposSet, nil
}

func getUnregisteredInputRepos(inputRepoList []string, alreadyRegisteredRepos sets.Set[string]) (
	unregisteredInputRepos []string, warnings []string) {
	if len(inputRepoList) > 0 {
		inputRepositoriesSet := sets.New(inputRepoList...)
		for inputRepo := range inputRepositoriesSet {
			// Input repos without owner are added to unregistered list, even if already registered
			if alreadyRegisteredRepos.Has(inputRepo) {
				warnings = append(warnings, fmt.Sprintf("Repository %s is already registered", inputRepo))
			} else {
				unregisteredInputRepos = append(unregisteredInputRepos, inputRepo)
			}
		}
	}
	return unregisteredInputRepos, warnings
}

func fetchRemoteRepositoriesFromProvider(ctx context.Context, provider, project string, client minderv1.RepositoryServiceClient) (
	[]*minderv1.UpstreamRepositoryRef, error) {
	var provPtr *string
	if provider != "" {
		provPtr = &provider
	}
	remoteListResp, err := client.ListRemoteRepositoriesFromProvider(ctx, &minderv1.ListRemoteRepositoriesFromProviderRequest{
		Context: &minderv1.Context{
			Provider: provPtr,
			Project:  &project,
		},
	})
	if err != nil {
		return nil, err
	}
	return remoteListResp.Results, nil
}

func getUnregisteredRemoteRepositories(remoteRepositories []*minderv1.UpstreamRepositoryRef,
	alreadyRegisteredRepos sets.Set[string]) []*minderv1.UpstreamRepositoryRef {
	var unregisteredRepos []*minderv1.UpstreamRepositoryRef
	for _, remoteRepo := range remoteRepositories {
		if !alreadyRegisteredRepos.Has(cli.GetRepositoryName(remoteRepo.Owner, remoteRepo.Name)) {
			unregisteredRepos = append(unregisteredRepos, &minderv1.UpstreamRepositoryRef{
				Owner:   remoteRepo.Owner,
				Name:    remoteRepo.Name,
				RepoId:  remoteRepo.RepoId,
				Context: remoteRepo.Context,
			})
		}
	}
	return unregisteredRepos
}

func getSelectedRepositories(repoList []*minderv1.UpstreamRepositoryRef, inputRepositories []string) (
	[]*minderv1.UpstreamRepositoryRef, []string, error) {
	// If no repos are found, exit
	if len(repoList) == 0 {
		return nil, nil, fmt.Errorf("no repositories found")
	}

	// Create a slice of strings to hold the repo names
	repoNames := make([]string, len(repoList))

	// Map of repo names to IDs
	repoIDs := make(map[string]int64)

	// Map of repo names to repo objects
	repoMap := make(map[string]*minderv1.UpstreamRepositoryRef)

	// Populate the repoNames slice, repoIDs map and repoMap
	for i, repo := range repoList {
		repoNames[i] = fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
		repoIDs[repoNames[i]] = repo.RepoId
		repoMap[repoNames[i]] = repo
	}

	// If the --name flag is set, use it to select repos
	allSelectedRepos, warnings := getSelectedInputRepositories(inputRepositories, repoIDs)

	// The repo flag was empty, or no repositories matched the ones from the flag
	// Prompt the user to select repos
	if len(allSelectedRepos) == 0 {
		var userSelectedRepos []string
		prompt := &survey.MultiSelect{
			Message: "Select repositories to register with Minder: \n",
			Options: repoNames,
		}
		// Prompt the user to select repos, defaulting to 20 per page, but scrollable
		err := survey.AskOne(prompt, &userSelectedRepos, survey.WithPageSize(20))
		if err != nil {
			return nil, warnings, fmt.Errorf("error getting repo selection: %s", err)
		}
		allSelectedRepos = append(allSelectedRepos, userSelectedRepos...)
	}

	// If no repos were selected, exit
	if len(allSelectedRepos) == 0 {
		return nil, warnings, fmt.Errorf("no repositories selected")
	}

	// Create a slice of Repositories protobufs
	protoRepos := make([]*minderv1.UpstreamRepositoryRef, len(allSelectedRepos))

	// Convert the selected repos into a slice of Repositories protobufs
	for i, repo := range allSelectedRepos {
		splitRepo := strings.Split(repo, "/")
		if len(splitRepo) != 2 {
			warnings = append(warnings, fmt.Sprintf("Unexpected repository name format: %s, skipping registration", repo))
			continue
		}
		protoRepos[i] = &minderv1.UpstreamRepositoryRef{
			Owner:  splitRepo[0],
			Name:   splitRepo[1],
			RepoId: repoIDs[repo],
			Context: &minderv1.Context{
				Provider: ptr.Ptr(repoMap[repo].GetContext().GetProvider()),
			},
		}
	}
	return protoRepos, warnings, nil
}

func registerSelectedRepos(
	project string,
	client minderv1.RepositoryServiceClient,
	selectedRepos []*minderv1.UpstreamRepositoryRef) ([]*minderv1.RegisterRepoResult, []string) {
	var results []*minderv1.RegisterRepoResult
	var warnings []string
	for idx := range selectedRepos {
		repo := selectedRepos[idx]

		result, err := client.RegisterRepository(context.Background(), &minderv1.RegisterRepositoryRequest{
			Context: &minderv1.Context{
				Provider: ptr.Ptr(repo.GetContext().GetProvider()),
				Project:  &project,
			},
			Repository: repo,
		})

		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Error registering repository %s: %s", repo.Name, err))
			continue
		}
		results = append(results, result.Result)
	}
	return results, warnings
}

func printRepoRegistrationStatus(cmd *cobra.Command, results []*minderv1.RegisterRepoResult) {
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

func getSelectedInputRepositories(inputRepositories []string, repoIDs map[string]int64) (selectedInputRepo, warnings []string) {
	for _, repo := range inputRepositories {
		if _, ok := repoIDs[repo]; !ok {
			warnings = append(warnings, fmt.Sprintf("Repository %s not found", repo))
			continue
		}
		selectedInputRepo = append(selectedInputRepo, repo)
	}
	return selectedInputRepo, warnings
}

func printWarnings(cmd *cobra.Command, warnings []string) {
	for _, warning := range warnings {
		cmd.Println(warning)
	}
}

func getInputRepoList(raw string) []string {
	if raw == "" {
		return []string{}
	}
	return strings.Split(raw, ",")
}

func init() {
	RepoCmd.AddCommand(repoRegisterCmd)
	// Flags
	repoRegisterCmd.Flags().StringSliceP("name", "n", []string{}, "List of repository names to register, i.e owner/repo,owner/repo")
}
