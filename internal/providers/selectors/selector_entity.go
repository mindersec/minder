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

// Package selectors provides the conversion of entities to SelectorEntities
package selectors

import (
	"context"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	"github.com/stacklok/minder/internal/entities/models"
	"github.com/stacklok/minder/internal/entities/properties"
	internalpb "github.com/stacklok/minder/internal/proto"
	ghprop "github.com/stacklok/minder/internal/providers/github/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

type toSelectorEntity func(ctx context.Context, entityWithProps *models.EntityWithProperties) *internalpb.SelectorEntity

func repoToSelectorEntity(
	_ context.Context, entityWithProps *models.EntityWithProperties) *internalpb.SelectorEntity {
	if entityWithProps.Entity.Type != minderv1.Entity_ENTITY_REPOSITORIES {
		return nil
	}

	var isFork *bool
	if propIsFork, err := entityWithProps.Properties.GetProperty(properties.RepoPropertyIsFork).AsBool(); err == nil {
		isFork = proto.Bool(propIsFork)
	}

	var isPrivate *bool
	if propIsPrivate, err := entityWithProps.Properties.GetProperty(properties.RepoPropertyIsPrivate).AsBool(); err == nil {
		isPrivate = proto.Bool(propIsPrivate)
	}

	return &internalpb.SelectorEntity{
		EntityType: minderv1.Entity_ENTITY_REPOSITORIES,
		Name:       entityWithProps.Entity.Name,
		Entity: &internalpb.SelectorEntity_Repository{
			Repository: &internalpb.SelectorRepository{
				Name:       entityWithProps.Entity.Name,
				IsFork:     isFork,
				IsPrivate:  isPrivate,
				Properties: entityWithProps.Properties.ToProtoStruct(),
			},
		},
	}
}

func artifactToSelectorEntity(
	ctx context.Context, entityWithProps *models.EntityWithProperties) *internalpb.SelectorEntity {
	if entityWithProps.Entity.Type != minderv1.Entity_ENTITY_ARTIFACTS {
		return nil
	}

	return &internalpb.SelectorEntity{
		EntityType: minderv1.Entity_ENTITY_ARTIFACTS,
		Name:       entityWithProps.Entity.Name,
		Entity: &internalpb.SelectorEntity_Artifact{
			Artifact: &internalpb.SelectorArtifact{
				Name:       entityWithProps.Entity.Name,
				Type:       entityWithProps.Properties.GetProperty(ghprop.ArtifactPropertyType).GetString(),
				Properties: entityWithProps.Properties.ToProtoStruct(),
			},
		},
	}
}

func pullRequestToSelectorEntity(
	_ context.Context, entityWithProps *models.EntityWithProperties) *internalpb.SelectorEntity {
	if entityWithProps.Entity.Type != minderv1.Entity_ENTITY_PULL_REQUESTS {
		return nil
	}

	return &internalpb.SelectorEntity{
		EntityType: minderv1.Entity_ENTITY_PULL_REQUESTS,
		Name:       entityWithProps.Entity.Name,
		Entity: &internalpb.SelectorEntity_PullRequest{
			PullRequest: &internalpb.SelectorPullRequest{
				Name:       entityWithProps.Entity.Name,
				Properties: entityWithProps.Properties.ToProtoStruct(),
			},
		},
	}
}

// newConverterFactory creates a new converterFactory with the default converters for each entity type
func newConverter(entType minderv1.Entity) toSelectorEntity {
	switch entType { // nolint:exhaustive
	case minderv1.Entity_ENTITY_REPOSITORIES:
		return repoToSelectorEntity
	case minderv1.Entity_ENTITY_ARTIFACTS:
		return artifactToSelectorEntity
	case minderv1.Entity_ENTITY_PULL_REQUESTS:
		return pullRequestToSelectorEntity
	}
	return nil
}

// EntityToSelectorEntity converts an entity to a SelectorEntity
func EntityToSelectorEntity(
	ctx context.Context,
	entType minderv1.Entity,
	entityWithProps *models.EntityWithProperties,
) *internalpb.SelectorEntity {
	converter := newConverter(entType)
	if converter == nil {
		zerolog.Ctx(ctx).Error().Str("entType", entType.ToString()).Msg("No converter available")
		return nil
	}
	return converter(ctx, entityWithProps)
}
