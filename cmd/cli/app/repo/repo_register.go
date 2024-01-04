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
	"os"
	"slices"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"

	"github.com/stacklok/minder/cmd/cli/app"
	"github.com/stacklok/minder/internal/util/cli"
	"github.com/stacklok/minder/internal/util/cli/table"
	"github.com/stacklok/minder/internal/util/cli/table/layouts"
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
func RegisterCmd(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	client := minderv1.NewRepositoryServiceClient(conn)

	provider := viper.GetString("provider")
	project := viper.GetString("project")
	repoList := viper.GetString("name")

	// Ensure provider is supported
	if !app.IsProviderSupported(provider) {
		return cli.MessageAndError(fmt.Sprintf("Provider %s is not supported yet", provider), fmt.Errorf("invalid argument"))
	}

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	// Get the list of repos
	listResp, err := client.ListRepositories(ctx, &minderv1.ListRepositoriesRequest{
		Context: &minderv1.Context{Provider: &provider, Project: &project},
		// keep this until we decide to delete them from the payload and rely only on the context
		Provider:  provider,
		ProjectId: project,
	})
	if err != nil {
		return cli.MessageAndError("Error getting list of repos", err)
	}

	// Get a list of remote repos
	remoteListResp, err := client.ListRemoteRepositoriesFromProvider(ctx, &minderv1.ListRemoteRepositoriesFromProviderRequest{
		Context: &minderv1.Context{Provider: &provider, Project: &project},
		// keep this until we decide to delete them from the payload and rely only on the context
		Provider:  provider,
		ProjectId: project,
	})
	if err != nil {
		return cli.MessageAndError("Error getting list of remote repos", err)
	}

	// Unregistered repos are in remoteListResp but not in listResp
	// build a list of unregistered repos
	var unregisteredRepos []*minderv1.UpstreamRepositoryRef
	for _, remoteRepo := range remoteListResp.Results {
		found := false
		for _, repo := range listResp.Results {
			if remoteRepo.Owner == repo.Owner && remoteRepo.Name == repo.Name {
				found = true
				break
			}
		}
		if !found {
			unregisteredRepos = append(unregisteredRepos, &minderv1.UpstreamRepositoryRef{
				Owner:  remoteRepo.Owner,
				Name:   remoteRepo.Name,
				RepoId: remoteRepo.RepoId,
			})
		}
	}

	cmd.Printf("Found %d remote repositories: %d registered and %d unregistered.\n",
		len(remoteListResp.Results), len(listResp.Results), len(unregisteredRepos))

	// Get the selected repos
	selectedRepos, err := getSelectedRepositories(unregisteredRepos, repoList)
	if err != nil {
		return cli.MessageAndError("Error getting selected repositories", err)
	}

	var results []*minderv1.RegisterRepoResult
	for idx := range selectedRepos {
		repo := selectedRepos[idx]

		result, err := client.RegisterRepository(context.Background(), &minderv1.RegisterRepositoryRequest{
			Context: &minderv1.Context{Provider: &provider, Project: &project},
			// keep this until we decide to delete them from the payload and rely only on the context
			Provider:   provider,
			ProjectId:  project,
			Repository: repo,
		})
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error registering repository %s: %s\n", repo.Name, err)
			continue
		}

		results = append(results, result.Result)
	}

	// Register the repos
	// The result gives a list of repositories with the registration status
	// Let's parse the results and print the status
	t := table.New(table.Simple, layouts.Default, []string{"Repository", "Status", "Message"})
	for _, result := range results {
		row := []string{fmt.Sprintf("%s/%s", result.Repository.Owner, result.Repository.Name)}
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
	return nil
}

func getSelectedRepositories(repoList []*minderv1.UpstreamRepositoryRef, flagRepos string) (
	[]*minderv1.UpstreamRepositoryRef, error) {
	// If no repos are found, exit
	if len(repoList) == 0 {
		return nil, fmt.Errorf("no repositories found")
	}

	// Create a slice of strings to hold the repo names
	repoNames := make([]string, len(repoList))

	// Map of repo names to IDs
	repoIDs := make(map[string]int32)

	// Populate the repoNames slice and repoIDs map
	for i, repo := range repoList {
		repoNames[i] = fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
		repoIDs[repoNames[i]] = repo.RepoId
	}

	// Create a slice of strings to hold the selected repos
	var allSelectedRepos []string

	// If the --repo flag is set, use it to select repos
	if flagRepos != "" {
		repos := strings.Split(flagRepos, ",")
		for _, repo := range repos {
			if !slices.Contains(repoNames, repo) {
				_, _ = fmt.Fprintf(os.Stderr, "Repository %s not found\n", repo)
				continue
			}
			allSelectedRepos = append(allSelectedRepos, repo)
		}
	}

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
			return nil, fmt.Errorf("error getting repo selection: %s", err)
		}
		allSelectedRepos = append(allSelectedRepos, userSelectedRepos...)
	}

	// If no repos were selected, exit
	if len(allSelectedRepos) == 0 {
		return nil, fmt.Errorf("no repositories selected")
	}

	// Create a slice of Repositories protobufs
	protoRepos := make([]*minderv1.UpstreamRepositoryRef, len(allSelectedRepos))

	// Convert the selected repos into a slice of Repositories protobufs
	for i, repo := range allSelectedRepos {
		splitRepo := strings.Split(repo, "/")
		if len(splitRepo) != 2 {
			_, _ = fmt.Fprintf(os.Stderr, "Unexpected repository name format: %s, skipping registration\n", repo)
			continue
		}
		protoRepos[i] = &minderv1.UpstreamRepositoryRef{
			Owner:  splitRepo[0],
			Name:   splitRepo[1],
			RepoId: repoIDs[repo],
		}
	}
	return protoRepos, nil
}

func init() {
	RepoCmd.AddCommand(repoRegisterCmd)
	// Flags
	repoRegisterCmd.Flags().StringP("name", "n", "", "List of repository names to register, i.e owner/repo,owner/repo")
}
