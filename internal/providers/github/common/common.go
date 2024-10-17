// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package common provides common utilities for the GitHub provider
package common

import (
	gogithub "github.com/google/go-github/v63/github"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/mindersec/minder/internal/providers/github/properties"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

// ConvertRepositories converts a list of GitHub repositories to a list of minder repositories
func ConvertRepositories(repos []*gogithub.Repository) []*minderv1.Repository {
	var converted []*minderv1.Repository
	for _, repo := range repos {
		// Skip archived repositories
		if repo.Archived != nil && *repo.Archived {
			continue
		}
		propsMap := properties.GitHubRepoToMap(repo)
		props, err := structpb.NewStruct(propsMap)
		if err != nil {
			continue
		}
		converted = append(converted, ConvertRepository(repo, props))
	}
	return converted
}

// ConvertRepository converts a GitHub repository to a minder repository
func ConvertRepository(repo *gogithub.Repository, props *structpb.Struct) *minderv1.Repository {
	return &minderv1.Repository{
		Name:       repo.GetName(),
		Owner:      repo.GetOwner().GetLogin(),
		RepoId:     repo.GetID(),
		HookUrl:    repo.GetHooksURL(),
		DeployUrl:  repo.GetDeploymentsURL(),
		CloneUrl:   repo.GetCloneURL(),
		IsPrivate:  *repo.Private,
		IsFork:     *repo.Fork,
		Properties: props,
	}
}
