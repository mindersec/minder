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

package gitlab

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/stacklok/minder/internal/entities/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// Repository Properties
const (
	// RepoPropertyProjectName represents the gitlab project
	RepoPropertyProjectName = "gitlab/project_name"
	// RepoPropertyDefaultBranch represents the gitlab default branch
	RepoPropertyDefaultBranch = "gitlab/default_branch"
	// RepoPropertyNamespace represents the gitlab repo namespace
	RepoPropertyNamespace = "gitlab/namespace"
	// RepoPropertyLicense represents the gitlab repo license
	RepoPropertyLicense = "gitlab/license"
	// RepoPropertyCloneURL represents the gitlab repo clone URL
	RepoPropertyCloneURL = "gitlab/clone_url"
	// RepoPropertyHookID represents the gitlab repo hook ID
	RepoPropertyHookID = "gitlab/hook_id"
	// RepoPropertyHookURL represents the gitlab repo hook URL
	RepoPropertyHookURL = "gitlab/hook_url"
)

// Pull Request Properties
const (
	// PullRequestProjectID represents the gitlab project ID
	PullRequestProjectID = "gitlab/project_id"
	// PullRequestNumber represents the gitlab merge request number
	PullRequestNumber = "gitlab/merge_request_number"
	// PullRequestSourceBranch represents the gitlab source branch
	PullRequestSourceBranch = "gitlab/source_branch"
	// PullRequestTargetBranch represents the gitlab target branch
	PullRequestTargetBranch = "gitlab/target_branch"
	// PullRequestAuthor represents the gitlab author
	PullRequestAuthor = "gitlab/author"
	// PullRequestCommitSHA represents the gitlab commit SHA
	PullRequestCommitSHA = "gitlab/commit_sha"
	// PullRequestURL represents the gitlab merge request URL
	PullRequestURL = "gitlab/merge_request_url"
)

// FetchAllProperties implements the provider interface
func (c *gitlabClient) FetchAllProperties(
	ctx context.Context, getByProps *properties.Properties, entType minderv1.Entity, _ *properties.Properties,
) (*properties.Properties, error) {
	if !c.SupportsEntity(entType) {
		return nil, fmt.Errorf("entity type %s not supported", entType)
	}

	switch entType {
	case minderv1.Entity_ENTITY_REPOSITORIES:
		return c.getPropertiesForRepo(ctx, getByProps)
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		return c.getPropertiesForPullRequest(ctx, getByProps)
	default:
		return nil, fmt.Errorf("entity type %s not supported", entType)
	}
}

// FetchProperty implements the provider interface
// TODO: Implement this
func (_ *gitlabClient) FetchProperty(
	_ context.Context, _ *properties.Properties, _ minderv1.Entity, _ string) (*properties.Property, error) {
	return nil, nil
}

// GetEntityName implements the provider interface
func (c *gitlabClient) GetEntityName(entityType minderv1.Entity, props *properties.Properties) (string, error) {
	if props == nil {
		return "", errors.New("properties are nil")
	}

	if !c.SupportsEntity(entityType) {
		return "", fmt.Errorf("entity type %s not supported", entityType)
	}

	switch entityType {
	case minderv1.Entity_ENTITY_REPOSITORIES:
		return c.getRepoNameFromProperties(props)
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		return c.getPullRequestNameFromProperties(props)
	default:
		return "", fmt.Errorf("entity type %s not supported", entityType)
	}
}

func (c *gitlabClient) getRepoNameFromProperties(props *properties.Properties) (string, error) {
	groupName, err := getStringProp(props, RepoPropertyNamespace)
	if err != nil {
		return "", err
	}

	projectName, err := getStringProp(props, RepoPropertyProjectName)
	if err != nil {
		return "", err
	}

	return formatRepoName(groupName, projectName), nil
}

func (c *gitlabClient) getPullRequestNameFromProperties(props *properties.Properties) (string, error) {
	groupName, err := getStringProp(props, RepoPropertyNamespace)
	if err != nil {
		return "", err
	}

	projectName, err := getStringProp(props, RepoPropertyProjectName)
	if err != nil {
		return "", err
	}

	iid, err := getStringProp(props, PullRequestNumber)
	if err != nil {
		return "", err
	}

	return formatPullRequestName(groupName, projectName, iid), nil
}

// PropertiesToProtoMessage implements the ProtoMessageConverter interface
func (c *gitlabClient) PropertiesToProtoMessage(
	entType minderv1.Entity, props *properties.Properties,
) (protoreflect.ProtoMessage, error) {
	if !c.SupportsEntity(entType) {
		return nil, fmt.Errorf("entity type %s is not supported by the gitlab provider", entType)
	}

	return repoV1FromProperties(props)
}

// FormatRepositoryUpstreamID returns the upstream ID for a gitlab project
// This is done so we don't have to deal with conversions in the provider
// when dealing with entities
func FormatRepositoryUpstreamID(id int) string {
	return fmt.Sprintf("%d", id)
}

// FormatPullRequestUpstreamID returns the upstream ID for a gitlab merge request
// This is done so we don't have to deal with conversions in the provider
// when dealing with entities
func FormatPullRequestUpstreamID(id int) string {
	return fmt.Sprintf("%d", id)
}

func getStringProp(props *properties.Properties, key string) (string, error) {
	value, err := props.GetProperty(key).AsString()
	if err != nil {
		return "", fmt.Errorf("property %s not found or not a string", key)
	}

	return value, nil
}

func formatRepoName(groupName, projectName string) string {
	return groupName + "/" + projectName
}

func formatPullRequestName(groupName, projectName, iid string) string {
	return fmt.Sprintf("%s/%s/%s", groupName, projectName, iid)
}
