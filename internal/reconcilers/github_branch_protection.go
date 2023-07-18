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

// Package reconcilers provides implementations of all the reconcilers
package reconcilers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	github "github.com/google/go-github/v53/github"
	"github.com/rs/zerolog/log"

	"github.com/stacklok/mediator/pkg/controlplane"
	"github.com/stacklok/mediator/pkg/db"
	ghclient "github.com/stacklok/mediator/pkg/providers/github"
)

// gojsonschema -p reconcilers pkg/controlplane/policy_types/github/branch_protection/1.0.0/schema.json

// SchemaJsonBranchesElem corresponds to the JSON schema field "branches.elem".
type SchemaJsonBranchesElem struct {
	// Name corresponds to the JSON schema field "name".
	Name string `json:"name" yaml:"name" mapstructure:"name"`

	// Rules corresponds to the JSON schema field "rules".
	Rules SchemaJsonBranchesElemRules `json:"rules" yaml:"rules" mapstructure:"rules"`
}

// SchemaJsonBranchesElemRules corresponds to the JSON schema field "branches.elem.rules".
// nolint: lll
type SchemaJsonBranchesElemRules struct {
	// Allows deletion of the protected branch by anyone with write access to the repository.
	AllowDeletions *bool `json:"allow_deletions,omitempty" yaml:"allow_deletions,omitempty" mapstructure:"allow_deletions,omitempty"`

	// Permits force pushes to the protected branch by anyone with write access to the repository.
	AllowForcePushes *bool `json:"allow_force_pushes,omitempty" yaml:"allow_force_pushes,omitempty" mapstructure:"allow_force_pushes,omitempty"`

	// Whether users can pull changes from upstream when the branch is locked. Set to
	// `true` to allow fork syncing. Set to `false` to prevent fork syncing.
	AllowForkSyncing *bool `json:"allow_fork_syncing,omitempty" yaml:"allow_fork_syncing,omitempty" mapstructure:"allow_fork_syncing,omitempty"`

	// Set to true to enforce required status checks for repository administrators
	EnforceAdmins *bool `json:"enforce_admins,omitempty" yaml:"enforce_admins,omitempty" mapstructure:"enforce_admins,omitempty"`

	// Whether to set the branch as read-only. If this is true, users will not be able to push to the branch.
	LockBranch *bool `json:"lock_branch,omitempty" yaml:"lock_branch,omitempty" mapstructure:"lock_branch,omitempty"`

	// Requires all conversations on code to be resolved before a pull request can be merged into a branch that matches this rule.
	RequiredConversationResolution *bool `json:"required_conversation_resolution,omitempty" yaml:"required_conversation_resolution,omitempty" mapstructure:"required_conversation_resolution,omitempty"`

	// Enforces a linear commit Git history, which prevents anyone from pushing merge commits to a branch.
	RequiredLinearHistory *bool `json:"required_linear_history,omitempty" yaml:"required_linear_history,omitempty" mapstructure:"required_linear_history,omitempty"`

	// RequiredPullRequestReviews corresponds to the JSON schema field
	// "required_pull_request_reviews".
	RequiredPullRequestReviews SchemaJsonBranchesElemRulesRequiredPullRequestReviews `json:"required_pull_request_reviews,omitempty" yaml:"required_pull_request_reviews,omitempty" mapstructure:"required_pull_request_reviews,omitempty"`

	// Wether this branch requires signed commits
	RequiredSignatures *bool `json:"required_signatures,omitempty" yaml:"required_signatures,omitempty" mapstructure:"required_signatures,omitempty"`
}

// SchemaJsonBranchesElemRulesRequiredPullRequestReviews corresponds to the
// JSON schema field "branches.elem.rules.required_pull_request_reviews".
type SchemaJsonBranchesElemRulesRequiredPullRequestReviews map[string]interface{}

// SchemaJson corresponds to the JSON schema field "branches".
type SchemaJson struct {
	// Branches corresponds to the JSON schema field "branches".
	Branches []SchemaJsonBranchesElem `json:"branches" yaml:"branches" mapstructure:"branches"`
}

// UnmarshalJSON implements json.Unmarshaler - needed for gojsonschema.
func (j *SchemaJson) unmarshalJSON(b []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if v, ok := raw["branches"]; !ok || v == nil {
		return fmt.Errorf("field branches in SchemaJson: required")
	}
	var plain SchemaJson
	if err := json.Unmarshal(b, &plain); err != nil {
		return err
	}
	*j = plain
	return nil
}

func getBranchProtection(ctx context.Context, store db.Store, groupId int32, owner string,
	repo string, branch string) (*github.Protection, error) {
	// Populate the database with the repositories using the GraphQL API
	token, err := controlplane.GetProviderAccessToken(ctx, store, ghclient.Github, groupId, false)
	if err != nil {
		return nil, err
	}
	client, err := ghclient.NewRestClient(ctx, ghclient.GitHubConfig{
		Token: token.AccessToken,
	})
	if err != nil {
		return nil, err
	}

	// first we need to query github API
	protection, err := client.GetBranchProtection(ctx, owner, repo, branch)
	if err != nil {
		githubErr := err.(*github.ErrorResponse)
		if githubErr.Response.StatusCode == 404 {
			// if not branch is found, just ignore it
			return nil, nil
		}
		return nil, err
	}
	return protection, nil
}

// nolint:gocyclo
func checkBranchDifferences(rules SchemaJsonBranchesElemRules, protection *github.Protection) []Differences {
	differences := make([]Differences, 0)

	// check and compare fields, appending differences found
	if rules.AllowDeletions != nil && *rules.AllowDeletions != protection.AllowDeletions.Enabled {
		differences = append(differences, Differences{Field: "AllowDeletions",
			ActualValue: protection.AllowDeletions.Enabled, ExpectedValue: rules.AllowDeletions})
	}
	if rules.AllowForcePushes != nil && *rules.AllowForcePushes != protection.AllowForcePushes.Enabled {
		differences = append(differences, Differences{Field: "AllowForcePushes",
			ActualValue: protection.AllowForcePushes.Enabled, ExpectedValue: rules.AllowForcePushes})
	}
	if rules.AllowForkSyncing != nil && rules.AllowForkSyncing != protection.AllowForkSyncing.Enabled {
		differences = append(differences, Differences{Field: "AllowForkSyncing",
			ActualValue: protection.AllowForkSyncing.Enabled, ExpectedValue: rules.AllowForkSyncing})
	}
	if rules.EnforceAdmins != nil && *rules.EnforceAdmins != protection.EnforceAdmins.Enabled {
		differences = append(differences, Differences{Field: "EnforceAdmins",
			ActualValue: protection.EnforceAdmins.Enabled, ExpectedValue: rules.EnforceAdmins})
	}
	if rules.LockBranch != nil && rules.LockBranch != protection.LockBranch.Enabled {
		differences = append(differences, Differences{Field: "LockBranch",
			ActualValue: protection.LockBranch.Enabled, ExpectedValue: rules.LockBranch})
	}
	if rules.RequiredConversationResolution != nil &&
		*rules.RequiredConversationResolution != protection.RequiredConversationResolution.Enabled {
		differences = append(differences, Differences{Field: "RequiredConversationResolution",
			ActualValue: protection.RequiredConversationResolution.Enabled, ExpectedValue: rules.RequiredConversationResolution})
	}
	if rules.RequiredLinearHistory != nil && *rules.RequiredLinearHistory != protection.RequireLinearHistory.Enabled {
		differences = append(differences, Differences{Field: "RequiredLinearHistory",
			ActualValue: protection.RequireLinearHistory.Enabled, ExpectedValue: rules.RequiredLinearHistory})
	}
	if rules.RequiredPullRequestReviews != nil && protection.RequiredPullRequestReviews == nil {
		differences = append(differences, Differences{Field: "RequiredPullRequestReviews",
			ActualValue: protection.RequiredPullRequestReviews, ExpectedValue: rules.RequiredPullRequestReviews})
	} else {
		if rules.RequiredPullRequestReviews != nil {
			if rules.RequiredPullRequestReviews["dismiss_stale_reviews"] != nil &&
				rules.RequiredPullRequestReviews["dismiss_stale_reviews"] != protection.RequiredPullRequestReviews.DismissStaleReviews {
				differences = append(differences, Differences{Field: "RequiredPullRequestReviews.dismiss_stale_reviews",
					ActualValue:   protection.RequiredPullRequestReviews.DismissStaleReviews,
					ExpectedValue: rules.RequiredPullRequestReviews["dismiss_stale_reviews"]})
			}
			if rules.RequiredPullRequestReviews["require_code_owner_reviews"] != nil &&
				rules.RequiredPullRequestReviews["require_code_owner_reviews"] !=
					protection.RequiredPullRequestReviews.RequireCodeOwnerReviews {
				differences = append(differences, Differences{Field: "RequiredPullRequestReviews.require_code_owner_reviews",
					ActualValue:   protection.RequiredPullRequestReviews.RequireCodeOwnerReviews,
					ExpectedValue: rules.RequiredPullRequestReviews["require_code_owner_reviews"]})
			}
			if rules.RequiredPullRequestReviews["required_approving_review_count"] != nil {
				if count, ok := rules.RequiredPullRequestReviews["required_approving_review_count"].(float64); ok {
					if int(count) != protection.RequiredPullRequestReviews.RequiredApprovingReviewCount {
						differences = append(differences, Differences{Field: "RequiredPullRequestReviews.required_approving_review_count",
							ActualValue:   protection.RequiredPullRequestReviews.RequiredApprovingReviewCount,
							ExpectedValue: rules.RequiredPullRequestReviews["required_approving_review_count"]})
					}
				}
			}
			if rules.RequiredPullRequestReviews["require_last_push_approval"] != nil &&
				rules.RequiredPullRequestReviews["require_last_push_approval"] !=
					protection.RequiredPullRequestReviews.RequireLastPushApproval {
				differences = append(differences, Differences{Field: "RequiredPullRequestReviews.require_last_push_approval",
					ActualValue:   protection.RequiredPullRequestReviews.RequireLastPushApproval,
					ExpectedValue: rules.RequiredPullRequestReviews["require_last_push_approval"]})
			}
		}
	}
	if rules.RequiredSignatures != nil && rules.RequiredSignatures != protection.RequiredSignatures.Enabled {
		differences = append(differences, Differences{Field: "RequiredSignatures",
			ActualValue: protection.RequiredSignatures.Enabled, ExpectedValue: rules.RequiredSignatures})
	}

	return differences
}

// ParseBranchProtectionEventGithub parses a branch protection event from GitHub
// nolint:gocyclo
func ParseBranchProtectionEventGithub(ctx context.Context, store db.Store, msg *message.Message) error {
	var event github.BranchProtectionRuleEvent
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		return err
	}

	// check policies for that repo
	policies, err := store.GetPoliciesByRepoAndType(ctx, db.GetPoliciesByRepoAndTypeParams{
		Provider: ghclient.Github, PolicyType: "branch_protection", RepoID: int32(*event.Rule.RepositoryID)})
	if err != nil {
		return err
	}

	if len(policies) == 0 {
		// no need to act, we do not have policies
		return nil
	}

	// check if repository exists
	repo, err := store.GetRepositoryByRepoID(ctx, db.GetRepositoryByRepoIDParams{Provider: ghclient.Github,
		RepoID: int32(*event.Rule.RepositoryID)})
	if err != nil {
		return err
	}

	// reconcile branch protection
	for _, policy := range policies {
		branch, err := getBranchProtection(ctx, store, policy.GroupID, *event.Repo.Owner.Login, *event.Repo.Name, *event.Rule.Name)
		if err != nil {
			return err
		}

		// if we have protection information for the branch, use it
		if branch != nil {
			// read policy and extract the relevant branch content
			var policyData SchemaJson
			if err := policyData.unmarshalJSON(policy.PolicyDefinition); err != nil {
				return err
			}

			// find the right branch
			for _, targetBranch := range policyData.Branches {
				if targetBranch.Name == *event.Rule.Name {
					differences := checkBranchDifferences(targetBranch.Rules, branch)

					// store differences
					if len(differences) > 0 {
						log.Info().Msgf("Policy violated for %s/%s:%s", *event.Repo.Owner.Login, *event.Repo.Name, *event.Rule.Name)

						// inform about policy status and violation
						err := store.UpdatePolicyStatus(ctx, db.UpdatePolicyStatusParams{RepositoryID: repo.ID,
							PolicyID: policy.ID, PolicyStatus: db.PolicyStatusTypesFailure})
						if err != nil {
							return err
						}
						result, err := json.MarshalIndent(differences, "", "  ")
						if err != nil {
							return err
						}

						metadata := map[string]interface{}{
							"branch":           *event.Rule.Name,
							"repository_id":    *event.Rule.RepositoryID,
							"repository_owner": *event.Repo.Owner.Login,
							"repository_name":  *event.Repo.Name,
						}
						mresult, err := json.MarshalIndent(metadata, "", "  ")
						if err != nil {
							return err
						}
						_, err = store.CreatePolicyViolation(ctx, db.CreatePolicyViolationParams{RepositoryID: repo.ID,
							PolicyID: policy.ID, Metadata: json.RawMessage(mresult), Violation: json.RawMessage(result)})
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}
