//
// Copyright 2024 Stacklok, Inc.
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

package artifact

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
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register an artifact",
	Long:  `The repo register subcommand is used to register an artifact within Minder.`,
	RunE:  cli.GRPCClientWrapRunE(RegisterCmd),
}

// RegisterCmd represents the register command to register an artifact with minder
//
//nolint:gocyclo
func RegisterCmd(ctx context.Context, cmd *cobra.Command, conn *grpc.ClientConn) error {
	client := minderv1.NewArtifactServiceClient(conn)

	provider := viper.GetString("provider")
	project := viper.GetString("project")
	inputRepoList := viper.GetString("name")

	// No longer print usage on returned error, since we've parsed our inputs
	// See https://github.com/spf13/cobra/issues/340#issuecomment-374617413
	cmd.SilenceUsage = true

	alreadyRegistered, err := fetchAlreadyRegistered(ctx, provider, project, client)
	if err != nil {
		return cli.MessageAndError("Error getting list of registered repos", err)
	}

	unregisteredInput, warnings := getUnregisteredInput(inputRepoList, alreadyRegistered)
	printWarnings(cmd, warnings)

	// All input repos are already registered
	if inputRepoList != "" && len(unregisteredInput) == 0 {
		return nil
	}

	remoteArts, err := fetchRemoteArtifactsFromProvider(ctx, provider, project, client)
	if err != nil {
		return cli.MessageAndError("Error getting list of remote repos", err)
	}

	unregisteredRemoteArtifacts := getUnregisteredRemoteArtifacts(remoteArts, alreadyRegistered)

	cmd.Printf("Found %d remote repositories: %d registered and %d unregistered.\n",
		len(remoteArts), len(alreadyRegistered), len(unregisteredRemoteArtifacts))

	selected, warnings, err := getSelectedArtifacts(unregisteredRemoteArtifacts, unregisteredInput)
	if err != nil {
		return cli.MessageAndError("Error getting selected repositories", err)
	}
	printWarnings(cmd, warnings)

	results, warnings := registerSelected(provider, project, client, selected)
	printWarnings(cmd, warnings)

	printRegistered(cmd, results)
	return nil
}

func fetchAlreadyRegistered(ctx context.Context, provider, project string, client minderv1.ArtifactServiceClient) (
	sets.Set[string], error) {
	alreadyRegistered, err := client.ListArtifacts(ctx, &minderv1.ListArtifactsRequest{
		Context: &minderv1.Context{Provider: &provider, Project: &project},
	})
	if err != nil {
		return nil, err
	}

	alreadyRegisteredSet := sets.New[string]()
	for _, art := range alreadyRegistered.Results {
		alreadyRegisteredSet.Insert(art.Name)
	}

	return alreadyRegisteredSet, nil
}

func getUnregisteredInput(inputList string, alreadyRegistered sets.Set[string]) (
	unregisteredInput []string, warnings []string) {
	if inputList != "" {
		inputSlice := strings.Split(inputList, ",")
		inputitoriesSet := sets.New(inputSlice...)
		for inputArt := range inputitoriesSet {
			// Input repos without owner are added to unregistered list, even if already registered
			if alreadyRegistered.Has(inputArt) {
				warnings = append(warnings, fmt.Sprintf("artifact %s is already registered", inputArt))
			} else {
				unregisteredInput = append(unregisteredInput, inputArt)
			}
		}
	}
	return unregisteredInput, warnings
}

func fetchRemoteArtifactsFromProvider(ctx context.Context, provider, project string, client minderv1.ArtifactServiceClient) (
	[]*minderv1.UpstreamArtifactRef, error) {
	remoteListResp, err := client.ListRemoteArtifactsFromProvider(ctx, &minderv1.ListRemoteArtifactsFromProviderRequest{
		Context: &minderv1.Context{Provider: &provider, Project: &project},
		Type:    "container",
	})
	if err != nil {
		return nil, err
	}
	return remoteListResp.Results, nil
}

func getUnregisteredRemoteArtifacts(
	remotearts []*minderv1.UpstreamArtifactRef,
	alreadyRegistered sets.Set[string],
) []*minderv1.UpstreamArtifactRef {
	var unregistered []*minderv1.UpstreamArtifactRef
	for _, remart := range remotearts {
		if !alreadyRegistered.Has(remart.Name) {
			unregistered = append(unregistered, &minderv1.UpstreamArtifactRef{
				Name: remart.Name,
				Type: "container",
			})
		}
	}
	return unregistered
}

func getSelectedArtifacts(
	artifactList []*minderv1.UpstreamArtifactRef,
	inputarts []string,
) ([]*minderv1.UpstreamArtifactRef, []string, error) {
	// If no repos are found, exit
	if len(artifactList) == 0 {
		return nil, nil, fmt.Errorf("no repositories found")
	}

	// Create a slice of strings to hold the repo names
	artNames := make([]string, len(artifactList))

	// Populate the repoNames slice and repoIDs map
	for i, art := range artifactList {
		artNames[i] = art.Name
	}

	// If the --name flag is set, use it to select repos
	allSelected, warnings := getSelectedInputArtifacts(inputarts)

	// The repo flag was empty, or no repositories matched the ones from the flag
	// Prompt the user to select repos
	if len(allSelected) == 0 {
		var userSelected []string
		prompt := &survey.MultiSelect{
			Message: "Select artifacts to register with Minder: \n",
			Options: artNames,
		}
		// Prompt the user to select repos, defaulting to 20 per page, but scrollable
		err := survey.AskOne(prompt, &userSelected, survey.WithPageSize(20))
		if err != nil {
			return nil, warnings, fmt.Errorf("error getting repo selection: %s", err)
		}
		allSelected = append(allSelected, userSelected...)
	}

	// If no repos were selected, exit
	if len(allSelected) == 0 {
		return nil, warnings, fmt.Errorf("no repositories selected")
	}

	// Create a slice of itories protobufs
	proto := make([]*minderv1.UpstreamArtifactRef, len(allSelected))

	// Convert the selected repos into a slice of itories protobufs
	for i, art := range allSelected {
		proto[i] = &minderv1.UpstreamArtifactRef{
			Name: art,
			Type: "container",
		}
	}
	return proto, warnings, nil
}

func registerSelected(
	provider, project string,
	client minderv1.ArtifactServiceClient,
	selected []*minderv1.UpstreamArtifactRef,
) ([]*minderv1.Artifact, []string) {
	var results []*minderv1.Artifact
	var warnings []string
	for idx := range selected {
		art := selected[idx]

		result, err := client.RegisterArtifact(context.Background(), &minderv1.RegisterArtifactRequest{
			Context:  &minderv1.Context{Provider: &provider, Project: &project},
			Artifact: art,
		})

		if err != nil {
			warnings = append(warnings, fmt.Sprintf("Error registering artifact %s: %s", art.Name, err))
		} else {
			results = append(results, result.Artifact)
		}
	}
	return results, warnings
}

func printRegistered(cmd *cobra.Command, results []*minderv1.Artifact) {
	t := table.New(table.Simple, layouts.Default, []string{"Name", "Message"})
	for _, result := range results {
		// in the case of a malformed response, skip over it to avoid segfaulting
		if result == nil {
			cmd.Printf("Skipping malformed response: %v", result)
		}
		row := []string{result.Name}
		t.AddRow(row...)
	}
	t.Render()
}

func getSelectedInputArtifacts(input []string) (selectInputArtifact, warnings []string) {
	for _, repo := range input {
		selectInputArtifact = append(selectInputArtifact, repo)
	}
	return selectInputArtifact, warnings
}

func printWarnings(cmd *cobra.Command, warnings []string) {
	for _, warning := range warnings {
		cmd.Println(warning)
	}
}

func init() {
	ArtifactCmd.AddCommand(registerCmd)
	// Flags
	registerCmd.Flags().StringP("name", "n", "", "List of artifact names to register")
}
