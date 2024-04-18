// Copyright 2024 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package common provides common utilities for the GitHub provider
package common

import (
	gogithub "github.com/google/go-github/v61/github"

	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// ConvertRepositories converts a list of GitHub repositories to a list of minder repositories
func ConvertRepositories(repos []*gogithub.Repository) []*minderv1.Repository {
	var converted []*minderv1.Repository
	for _, repo := range repos {
		// Skip archived repositories
		if repo.Archived != nil && *repo.Archived {
			continue
		}
		converted = append(converted, ConvertRepository(repo))
	}
	return converted
}

// ConvertRepository converts a GitHub repository to a minder repository
func ConvertRepository(repo *gogithub.Repository) *minderv1.Repository {
	return &minderv1.Repository{
		Name:      repo.GetName(),
		Owner:     repo.GetOwner().GetLogin(),
		RepoId:    repo.GetID(),
		HookUrl:   repo.GetHooksURL(),
		DeployUrl: repo.GetDeploymentsURL(),
		CloneUrl:  repo.GetCloneURL(),
		IsPrivate: *repo.Private,
		IsFork:    *repo.Fork,
	}
}
