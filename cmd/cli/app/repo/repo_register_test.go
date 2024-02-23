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

package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	"k8s.io/apimachinery/pkg/util/sets"

	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func TestMessageJsonEncoding(t *testing.T) {
	t.Parallel()
	jsonMessage := `{
		"repository":{"owner":"test","name":"a-test","repoId":4000000000},
		"context":{"provider":"github","project":"1234"}
	}`

	protoMessage := minderv1.RegisterRepositoryRequest{}
	if err := protojson.Unmarshal([]byte(jsonMessage), &protoMessage); err != nil {
		t.Fatalf("Failed to unmarshal json message: %v", err)
	}

	if protoMessage.GetRepository().GetRepoId() != 4000000000 {
		t.Fatalf("Failed to unmarshal repoId, got %d", protoMessage.GetRepository().GetRepoId())
	}
}

func TestGetUnregisteredInputRepos(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		inputRepositories      string // comma separated list of repos
		alreadyRegisteredRepos sets.Set[string]
		unregisteredInputRepos []string
	}{
		{
			name:                   "empty repos",
			inputRepositories:      "",
			alreadyRegisteredRepos: sets.Set[string]{},
			unregisteredInputRepos: []string{},
		},
		{
			name:                   "no registered repos",
			inputRepositories:      "owner1/repo1,owner2/repo2",
			alreadyRegisteredRepos: sets.Set[string]{},
			unregisteredInputRepos: []string{"owner1/repo1", "owner2/repo2"},
		},
		{
			name:                   "no input repos",
			inputRepositories:      "",
			alreadyRegisteredRepos: sets.Set[string]{"owner1/repo1": {}, "owner2/repo2": {}},
			unregisteredInputRepos: []string{},
		},
		{
			name:                   "some registered repos",
			inputRepositories:      "owner1/repo1,owner2/repo2",
			alreadyRegisteredRepos: sets.Set[string]{"owner1/repo1": {}},
			unregisteredInputRepos: []string{"owner2/repo2"},
		},
		{
			name:                   "all registered repos",
			inputRepositories:      "owner1/repo1,owner2/repo2",
			alreadyRegisteredRepos: sets.Set[string]{"owner1/repo1": {}, "owner2/repo2": {}},
			unregisteredInputRepos: []string{},
		},
		{
			name:                   "some repos without owner",
			inputRepositories:      "owner1/repo1,owner2/repo2,repo3",
			alreadyRegisteredRepos: sets.Set[string]{"owner1/repo1": {}, "owner2/repo2": {}},
			unregisteredInputRepos: []string{"repo3"},
		},
		{
			name:                   "same name repo without owner",
			inputRepositories:      "owner1/repo1,owner2/repo2,repo2",
			alreadyRegisteredRepos: sets.Set[string]{"owner1/repo1": {}, "owner2/repo2": {}},
			unregisteredInputRepos: []string{"repo2"},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			unregisteredInputRepos, _ := getUnregisteredInputRepos(test.inputRepositories, test.alreadyRegisteredRepos)
			if len(unregisteredInputRepos) != len(test.unregisteredInputRepos) {
				t.Errorf("getUnregisteredInputRepos() = %v, unregisteredInputRepos %v", unregisteredInputRepos, test.unregisteredInputRepos)
			}
			for _, unregisteredInputRepo := range unregisteredInputRepos {
				if test.alreadyRegisteredRepos.Has(unregisteredInputRepo) {
					t.Errorf("getUnregisteredInputRepos() = %v, unregisteredInputRepos %v", unregisteredInputRepos, test.unregisteredInputRepos)
				}
			}
		})
	}
}

func TestGetUnregisteredRemoteRepositories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                      string
		remoteRepositories        []*minderv1.UpstreamRepositoryRef
		alreadyRegisteredRepos    sets.Set[string]
		expectedUnregisteredRepos []*minderv1.UpstreamRepositoryRef
	}{
		{
			name: "All remote repositories are unregistered",
			remoteRepositories: []*minderv1.UpstreamRepositoryRef{
				{Owner: "owner1", Name: "repo1", RepoId: 1},
				{Owner: "owner2", Name: "repo2", RepoId: 2},
			},
			alreadyRegisteredRepos: sets.Set[string]{},
			expectedUnregisteredRepos: []*minderv1.UpstreamRepositoryRef{
				{Owner: "owner1", Name: "repo1", RepoId: 1},
				{Owner: "owner2", Name: "repo2", RepoId: 2},
			},
		},
		{
			name: "Some remote repositories are already registered",
			remoteRepositories: []*minderv1.UpstreamRepositoryRef{
				{Owner: "owner1", Name: "repo1", RepoId: 1},
				{Owner: "owner2", Name: "repo2", RepoId: 2},
			},
			alreadyRegisteredRepos: sets.Set[string]{"owner1/repo1": {}},
			expectedUnregisteredRepos: []*minderv1.UpstreamRepositoryRef{
				{Owner: "owner2", Name: "repo2", RepoId: 2},
			},
		},
		{
			name: "All remote repositories are already registered",
			remoteRepositories: []*minderv1.UpstreamRepositoryRef{
				{Owner: "owner1", Name: "repo1", RepoId: 1},
				{Owner: "owner2", Name: "repo2", RepoId: 2},
			},
			alreadyRegisteredRepos:    sets.Set[string]{"owner1/repo1": {}, "owner2/repo2": {}},
			expectedUnregisteredRepos: []*minderv1.UpstreamRepositoryRef{},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			unregisteredRepos := getUnregisteredRemoteRepositories(test.remoteRepositories, test.alreadyRegisteredRepos)
			if len(unregisteredRepos) != len(test.expectedUnregisteredRepos) {
				t.Errorf("getUnregisteredRemoteRepositories() = %v, expected %v", unregisteredRepos, test.expectedUnregisteredRepos)
			}
			for i, repo := range unregisteredRepos {
				if test.expectedUnregisteredRepos[i] == nil ||
					repo.Owner != test.expectedUnregisteredRepos[i].Owner ||
					repo.Name != test.expectedUnregisteredRepos[i].Name ||
					repo.RepoId != test.expectedUnregisteredRepos[i].RepoId {
					t.Errorf("getUnregisteredRemoteRepositories() = %v, expected %v", repo, test.expectedUnregisteredRepos[i])
				}
			}
		})
	}
}

func TestGetSelectedInputRepositories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		inputRepositories     []string
		repoIDs               map[string]int64
		expectedSelectedRepos []string
		expectedWarnings      []string
	}{
		{
			name:              "All input repositories are selected",
			inputRepositories: []string{"owner1/repo1", "owner2/repo2"},
			repoIDs: map[string]int64{
				"owner1/repo1": 1,
				"owner2/repo2": 2,
			},
			expectedSelectedRepos: []string{"owner1/repo1", "owner2/repo2"},
			expectedWarnings:      nil,
		},
		{
			name:              "Some input repositories are not found",
			inputRepositories: []string{"owner1/repo1", "owner3/repo3"},
			repoIDs: map[string]int64{
				"owner1/repo1": 1,
				"owner2/repo2": 2,
			},
			expectedSelectedRepos: []string{"owner1/repo1"},
			expectedWarnings:      []string{"Repository owner3/repo3 not found"},
		},
		{
			name:              "No input repositories are found",
			inputRepositories: []string{"owner3/repo3", "owner4/repo4"},
			repoIDs: map[string]int64{
				"owner1/repo1": 1,
				"owner2/repo2": 2,
			},
			expectedSelectedRepos: nil,
			expectedWarnings:      []string{"Repository owner3/repo3 not found", "Repository owner4/repo4 not found"},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			selectedRepos, warnings := getSelectedInputRepositories(test.inputRepositories, test.repoIDs)
			assert.Equal(t, test.expectedSelectedRepos, selectedRepos)
			assert.Equal(t, test.expectedWarnings, warnings)
		})
	}
}
