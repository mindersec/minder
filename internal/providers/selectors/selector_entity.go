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
	"fmt"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/proto"

	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/entities/models"
	"github.com/stacklok/minder/internal/entities/properties"
	internalpb "github.com/stacklok/minder/internal/proto"
	ghprop "github.com/stacklok/minder/internal/providers/github/properties"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
)

func buildBaseSelectorEntity(
	entityWithProps *models.EntityWithProperties, selProv *internalpb.SelectorProvider) *internalpb.SelectorEntity {
	return &internalpb.SelectorEntity{
		EntityType: entityWithProps.Entity.Type,
		Name:       entityWithProps.Entity.Name,
		Provider:   selProv,
	}
}

type toSelectorEntity func(
	entityWithProps *models.EntityWithProperties, selProv *internalpb.SelectorProvider,
) *internalpb.SelectorEntity

func repoToSelectorEntity(
	entityWithProps *models.EntityWithProperties, selProv *internalpb.SelectorProvider,
) *internalpb.SelectorEntity {
	var isFork *bool
	if propIsFork, err := entityWithProps.Properties.GetProperty(properties.RepoPropertyIsFork).AsBool(); err == nil {
		isFork = proto.Bool(propIsFork)
	}

	var isPrivate *bool
	if propIsPrivate, err := entityWithProps.Properties.GetProperty(properties.RepoPropertyIsPrivate).AsBool(); err == nil {
		isPrivate = proto.Bool(propIsPrivate)
	}

	selEnt := buildBaseSelectorEntity(entityWithProps, selProv)
	selEnt.Entity = &internalpb.SelectorEntity_Repository{
		Repository: &internalpb.SelectorRepository{
			Name:       entityWithProps.Entity.Name,
			IsFork:     isFork,
			IsPrivate:  isPrivate,
			Properties: entityWithProps.Properties.ToProtoStruct(),
			Provider:   selProv,
		},
	}
	return selEnt
}

func artifactToSelectorEntity(
	entityWithProps *models.EntityWithProperties, selProv *internalpb.SelectorProvider,
) *internalpb.SelectorEntity {
	var artifactType string
	var err error
	artifactType, err = entityWithProps.Properties.GetProperty(properties.ArtifactPropertyType).AsString()
	if err != nil {
		artifactType = entityWithProps.Properties.GetProperty(ghprop.ArtifactPropertyType).GetString()
	}

	selEnt := buildBaseSelectorEntity(entityWithProps, selProv)
	selEnt.Entity = &internalpb.SelectorEntity_Artifact{
		Artifact: &internalpb.SelectorArtifact{
			Name:       entityWithProps.Entity.Name,
			Type:       artifactType,
			Properties: entityWithProps.Properties.ToProtoStruct(),
			Provider:   selProv,
		},
	}
	return selEnt
}

func pullRequestToSelectorEntity(
	entityWithProps *models.EntityWithProperties, selProv *internalpb.SelectorProvider,
) *internalpb.SelectorEntity {
	selEnt := buildBaseSelectorEntity(entityWithProps, selProv)
	selEnt.Entity = &internalpb.SelectorEntity_PullRequest{
		PullRequest: &internalpb.SelectorPullRequest{
			Name:       entityWithProps.Entity.Name,
			Properties: entityWithProps.Properties.ToProtoStruct(),
			Provider:   selProv,
		},
	}
	return selEnt
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

func fillProviderInfo(
	ctx context.Context,
	querier db.Store,
	entityWithProps *models.EntityWithProperties,
) (*internalpb.SelectorProvider, error) {
	if querier == nil {
		zerolog.Ctx(ctx).Warn().Msg("No querier, will not fill provider information")
		return nil, nil
	}

	dbProv, err := querier.GetProviderByID(ctx, entityWithProps.Entity.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider %s by ID", entityWithProps.Entity.ProviderID.String())
	}

	return &internalpb.SelectorProvider{
		Name:  dbProv.Name,
		Class: string(dbProv.Class),
	}, nil
}

// EntityToSelectorEntity converts an entity to a SelectorEntity
func EntityToSelectorEntity(
	ctx context.Context,
	querier db.Store,
	entType minderv1.Entity,
	entityWithProps *models.EntityWithProperties,
) *internalpb.SelectorEntity {
	converter := newConverter(entType)
	if converter == nil {
		zerolog.Ctx(ctx).Error().Str("entType", entType.ToString()).Msg("No converter available")
		return nil
	}

	selProv, err := fillProviderInfo(ctx, querier, entityWithProps)
	if err != nil {
		zerolog.Ctx(ctx).Error().
			Str("providerID", entityWithProps.Entity.ProviderID.String()).
			Err(err).
			Msg("Cannot fill provider information")
		return nil
	}
	selEnt := converter(entityWithProps, selProv)
	return selEnt
}
