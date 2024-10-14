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

	"github.com/mindersec/minder/internal/entities/properties"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
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

// Release Properties
const (
	// ReleasePropertyProjectID represents the gitlab project ID
	ReleasePropertyProjectID = "gitlab/project_id"
	// ReleasePropertyTag represents the gitlab release tag name.
	// NOTE: This is used for release discovery, not for creating releases.
	ReleasePropertyTag = "gitlab/tag"
	// ReleasePropertyBranch represents the gitlab release branch
	ReleasePropertyBranch = "gitlab/branch"
)

// FetchAllProperties implements the provider interface
func (c *gitlabClient) FetchAllProperties(
	ctx context.Context, getByProps *properties.Properties, entType minderv1.Entity, _ *properties.Properties,
) (*properties.Properties, error) {
	if !c.SupportsEntity(entType) {
		return nil, fmt.Errorf("entity type %s not supported", entType)
	}

	//nolint:exhaustive // We only support two entity types for now.
	switch entType {
	case minderv1.Entity_ENTITY_REPOSITORIES:
		return c.getPropertiesForRepo(ctx, getByProps)
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		return c.getPropertiesForPullRequest(ctx, getByProps)
	case minderv1.Entity_ENTITY_RELEASE:
		return c.getPropertiesForRelease(ctx, getByProps)
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

	//nolint:exhaustive // We only support two entity types for now.
	switch entityType {
	case minderv1.Entity_ENTITY_REPOSITORIES:
		return getRepoNameFromProperties(props)
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		return getPullRequestNameFromProperties(props)
	case minderv1.Entity_ENTITY_RELEASE:
		return getReleaseNameFromProperties(props)
	default:
		return "", fmt.Errorf("entity type %s not supported", entityType)
	}
}

// PropertiesToProtoMessage implements the ProtoMessageConverter interface
func (c *gitlabClient) PropertiesToProtoMessage(
	entType minderv1.Entity, props *properties.Properties,
) (protoreflect.ProtoMessage, error) {
	if !c.SupportsEntity(entType) {
		return nil, fmt.Errorf("entity type %s is not supported by the gitlab provider", entType)
	}

	//nolint:exhaustive // We only support two entity types for now.
	switch entType {
	case minderv1.Entity_ENTITY_REPOSITORIES:
		return repoV1FromProperties(props)
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		return pullRequestV1FromProperties(props)
	case minderv1.Entity_ENTITY_RELEASE:
		return releaseEntityV1FromProperties(props)
	default:
		return nil, fmt.Errorf("entity type %s not supported", entType)
	}
}

func getStringProp(props *properties.Properties, key string) (string, error) {
	value, err := props.GetProperty(key).AsString()
	if err != nil {
		return "", fmt.Errorf("property %s not found or not a string", key)
	}

	return value, nil
}
