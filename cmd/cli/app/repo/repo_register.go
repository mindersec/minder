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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package repo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/utils/strings/slices"

	github "github.com/stacklok/mediator/internal/providers/github"
	"github.com/stacklok/mediator/internal/util"
	"github.com/stacklok/mediator/internal/util/cli"
	pb "github.com/stacklok/mediator/pkg/api/protobuf/go/minder/v1"
)

var errNoRepositoriesSelected = errors.New("No repositories selected")
var cfgFlagRepos string

// repo_registerCmd represents the register command to register a repo with the
// minder control plane
var repo_registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a repo with the minder control plane",
	Long:  `Repo register is used to register a repo with the minder control plane`,
	PreRun: func(cmd *cobra.Command, args []string) {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			fmt.Fprintf(os.Stderr, "Error binding flags: %s\n", err)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		provider := util.GetConfigValue(viper.GetViper(), "provider", "provider", cmd, "").(string)
		if provider != github.Github {
			fmt.Fprintf(os.Stderr, "Only %s is supported at this time\n", github.Github)
			os.Exit(1)
		}
		projectID := viper.GetString("project-id")

		conn, err := util.GrpcForCommand(cmd, viper.GetViper())
		util.ExitNicelyOnError(err, "Error getting grpc connection")
		defer conn.Close()

		client := pb.NewRepositoryServiceClient(conn)
		ctx, cancel := util.GetAppContext()
		defer cancel()

		// Get the list of repos
		listResp, err := client.ListRepositories(ctx, &pb.ListRepositoriesRequest{
			Provider:  provider,
			ProjectId: projectID,
		})
		if err != nil {
			cli.PrintCmd(cmd, "Error getting list of repos: %s\n", err)
			os.Exit(1)
		}

		// Get a list of remote repos
		remoteListResp, err := client.ListRemoteRepositoriesFromProvider(ctx, &pb.ListRemoteRepositoriesFromProviderRequest{
			Provider:  provider,
			ProjectId: projectID,
		})
		if err != nil {
			cli.PrintCmd(cmd, "Error getting list of remote repos: %s\n", err)
			os.Exit(1)
		}

		// Unregistered repos are in remoteListResp but not in listResp
		// build a list of unregistered repos
		var unregisteredRepos []*pb.UpstreamRepositoryRef
		for _, remoteRepo := range remoteListResp.Results {
			found := false
			for _, repo := range listResp.Results {
				if remoteRepo.Owner == repo.Owner && remoteRepo.Name == repo.Name {
					found = true
					break
				}
			}
			if !found {
				unregisteredRepos = append(unregisteredRepos, &pb.UpstreamRepositoryRef{
					Owner:  remoteRepo.Owner,
					Name:   remoteRepo.Name,
					RepoId: remoteRepo.RepoId,
				})
			}
		}

		cli.PrintCmd(cmd, "Found %d remote repositories: %d registered and %d unregistered.\n",
			len(remoteListResp.Results), len(listResp.Results), len(unregisteredRepos))

		// Get the selected repos
		selectedRepos, err := getSelectedRepositories(unregisteredRepos, cfgFlagRepos)
		if err != nil {
			if errors.Is(err, errNoRepositoriesSelected) {
				_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
			} else {
				_, _ = fmt.Fprintf(os.Stderr, "Error getting selected repos: %s\n", err)
			}
			os.Exit(1)
		}

		results := []*pb.RegisterRepoResult{}
		for idx := range selectedRepos {
			repo := selectedRepos[idx]
			repoRegList := []*pb.UpstreamRepositoryRef{repo}
			// Construct the RegisterRepositoryRequest
			request := &pb.RegisterRepositoryRequest{
				Provider:     provider,
				Repositories: repoRegList,
				ProjectId:    projectID,
			}

			result, err := client.RegisterRepository(context.Background(), request)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Error registering repository %s: %s\n", repo.Name, err)
				continue
			}

			results = append(results, result.Results...)
		}

		// Register the repos
		// The result gives a list of repositories with the registration status
		// Let's parse the results and print the status
		columns := []table.Column{
			{Title: "Repository", Width: 35},
			{Title: "Status", Width: 15},
			{Title: "Message", Width: 60},
		}

		rows := make([]table.Row, len(results))
		for i, result := range results {
			rows[i] = table.Row{
				fmt.Sprintf("%s/%s", result.Repository.Owner, result.Repository.Name),
			}

			if result.Status.Success {
				rows[i] = append(rows[i], "Registered")
			} else {
				rows[i] = append(rows[i], "Failed")
			}

			if result.Status.Error != nil {
				rows[i] = append(rows[i], *result.Status.Error)
			} else {
				rows[i] = append(rows[i], "")
			}
		}

		t := table.New(
			table.WithColumns(columns),
			table.WithRows(rows),
			table.WithFocused(false),
			table.WithHeight(len(rows)),
			table.WithStyles(cli.TableHiddenSelectStyles),
		)

		cli.PrintCmd(cmd, cli.TableRender(t))
	},
}

func init() {
	RepoCmd.AddCommand(repo_registerCmd)
	repo_registerCmd.Flags().StringP("provider", "n", "", "Name for the provider to enroll")
	repo_registerCmd.Flags().StringP("project-id", "g", "", "ID of the project for repo registration")
	repo_registerCmd.Flags().StringVar(&cfgFlagRepos, "repo", "", "List of repositories to register, i.e owner/repo,owner/repo")
	if err := repo_registerCmd.MarkFlagRequired("provider"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error marking flag as required: %s\n", err)
	}
}

func getSelectedRepositories(repoList []*pb.UpstreamRepositoryRef, flagRepos string) ([]*pb.UpstreamRepositoryRef, error) {
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
		return nil, errNoRepositoriesSelected
	}

	// Create a slice of Repositories protobufs
	protoRepos := make([]*pb.UpstreamRepositoryRef, len(allSelectedRepos))

	// Convert the selected repos into a slice of Repositories protobufs
	for i, repo := range allSelectedRepos {
		splitRepo := strings.Split(repo, "/")
		if len(splitRepo) != 2 {
			_, _ = fmt.Fprintf(os.Stderr, "Unexpected repository name format: %s, skipping registration\n", repo)
			continue
		}
		protoRepos[i] = &pb.UpstreamRepositoryRef{
			Owner:  splitRepo[0],
			Name:   splitRepo[1],
			RepoId: repoIDs[repo],
		}
	}
	return protoRepos, nil
}
