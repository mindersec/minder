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

// Package events provides implementations of all the event handlers
// the GitHub provider supports.
package events

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stacklok/mediator/pkg/controlplane"
	"github.com/stacklok/mediator/pkg/db"
	"github.com/stacklok/mediator/pkg/providers/github"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

type GithubBranchProtectionEvent struct {
	Action string `json:"action"`
	Rule   struct {
		RepositoryID int32  `json:"repository_id"`
		Name         string `json:"name"`
	} `json:"rule"`
	Repository struct {
		FullName string `json:"full_name"`
	}
}

func getBranchProtection(ctx context.Context, store db.Store, groupId int32, policyDefinition json.RawMessage, event GithubBranchProtectionEvent) error {
	// Populate the database with the repositories using the GraphQL API
	token, err := controlplane.GetProviderAccessToken(ctx, store, github.Github, groupId)
	if err != nil {
		return err
	}

	client, err := ghclient.NewRestClient(ctx, ghclient.GitHubConfig{
		Token: token.AccessToken,
	})
	if err != nil {
		return err
	}

	// first we need to query github API
	protection, _, err := client.GetBranchProtection(ctx, owner, repo, branch)
	if err != nil {
		fmt.Printf("Error retrieving branch protection: %s\n", err)
		return
	}
}

func ParseBranchProtectionEventGithub(ctx context.Context, store db.Store, msg *message.Message) error {
	var event GithubBranchProtectionEvent
	err := json.Unmarshal([]byte(msg.Payload), &event)
	if err != nil {
		return err
	}

	// check policies for that repo
	policies, err := store.GetPoliciesByRepoAndType(ctx, db.GetPoliciesByRepoAndTypeParams{Provider: github.Github, PolicyType: "branch_protection", RepoID: event.Rule.RepositoryID})
	if err != nil {
		return err
	}

	if len(policies) == 0 {
		// no need to act, we do not have policies
		return nil
	}

	// reconcile branch protection
	for _, policy := range policies {
		branch := getBranchProtection(ctx, store, policy.GroupID, policy.PolicyDefinition, event)

	}

	return nil
}
