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

package github

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/stacklok/minder/internal/entities/models"
	"github.com/stacklok/minder/internal/entities/properties"
	internalpb "github.com/stacklok/minder/internal/proto"
	ghprop "github.com/stacklok/minder/internal/providers/github/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

// RepoToSelectorEntity converts a Repository to a SelectorEntity
func (_ *GitHub) RepoToSelectorEntity(
	_ context.Context, repoEntWithProps *models.EntityWithProperties,
) *internalpb.SelectorEntity {
	var isFork *bool
	if propIsFork, err := repoEntWithProps.Properties.GetProperty(properties.RepoPropertyIsFork).AsBool(); err == nil {
		isFork = proto.Bool(propIsFork)
	}

	var isPrivate *bool
	if propIsPrivate, err := repoEntWithProps.Properties.GetProperty(properties.RepoPropertyIsPrivate).AsBool(); err == nil {
		isPrivate = proto.Bool(propIsPrivate)
	}

	return &internalpb.SelectorEntity{
		EntityType: minderv1.Entity_ENTITY_REPOSITORIES,
		Name:       repoEntWithProps.Entity.Name,
		Entity: &internalpb.SelectorEntity_Repository{
			Repository: &internalpb.SelectorRepository{
				Name:       repoEntWithProps.Entity.Name,
				IsFork:     isFork,
				IsPrivate:  isPrivate,
				Properties: repoEntWithProps.Properties.ToProtoStruct(),
			},
		},
	}
}

// ArtifactToSelectorEntity converts an Artifact to a SelectorEntity
func (_ *GitHub) ArtifactToSelectorEntity(
	_ context.Context, artifactEntWithProps *models.EntityWithProperties,
) *internalpb.SelectorEntity {
	return &internalpb.SelectorEntity{
		EntityType: minderv1.Entity_ENTITY_ARTIFACTS,
		Name:       artifactEntWithProps.Entity.Name,
		Entity: &internalpb.SelectorEntity_Artifact{
			Artifact: &internalpb.SelectorArtifact{
				Name:       artifactEntWithProps.Entity.Name,
				Type:       artifactEntWithProps.Properties.GetProperty(ghprop.ArtifactPropertyType).GetString(),
				Properties: artifactEntWithProps.Properties.ToProtoStruct(),
			},
		},
	}
}

// PullRequestToSelectorEntity converts a Pull Request to a SelectorEntity
func (_ *GitHub) PullRequestToSelectorEntity(
	_ context.Context, pullRequestEntityWithProps *models.EntityWithProperties,
) *internalpb.SelectorEntity {
	return &internalpb.SelectorEntity{
		EntityType: minderv1.Entity_ENTITY_PULL_REQUESTS,
		Name:       pullRequestEntityWithProps.Entity.Name,
		Entity: &internalpb.SelectorEntity_PullRequest{
			PullRequest: &internalpb.SelectorPullRequest{
				Name:       pullRequestEntityWithProps.Entity.Name,
				Properties: pullRequestEntityWithProps.Properties.ToProtoStruct(),
			},
		},
	}
}
