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

package selectors

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	internalpb "github.com/stacklok/minder/internal/proto"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provifv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// entityInfoConverter is an interface for converting an entity from an EntityInfoWrapper to a SelectorEntity
type entityInfoConverter interface {
	toSelectorEntity(ctx context.Context, entity proto.Message) *internalpb.SelectorEntity
}

type repositoryInfoConverter struct {
	converter RepoSelectorConverter
}

func newRepositoryInfoConverter(provider provifv1.Provider) *repositoryInfoConverter {
	converter, err := provifv1.As[RepoSelectorConverter](provider)
	if err != nil {
		return nil
	}

	return &repositoryInfoConverter{
		converter: converter,
	}
}

func (rc *repositoryInfoConverter) toSelectorEntity(ctx context.Context, entity proto.Message) *internalpb.SelectorEntity {
	if rc == nil {
		return nil
	}

	r, ok := entity.(*minderv1.Repository)
	if !ok {
		return nil
	}

	return rc.converter.RepoToSelectorEntity(ctx, r)
}

type artifactInfoConverter struct {
	converter ArtifactSelectorConverter
}

func newArtifactInfoConverter(provider provifv1.Provider) *artifactInfoConverter {
	converter, err := provifv1.As[ArtifactSelectorConverter](provider)
	if err != nil {
		return nil
	}

	return &artifactInfoConverter{
		converter: converter,
	}
}

func (ac *artifactInfoConverter) toSelectorEntity(ctx context.Context, entity proto.Message) *internalpb.SelectorEntity {
	if ac == nil {
		return nil
	}

	a, ok := entity.(*minderv1.Artifact)
	if !ok {
		return nil
	}

	return ac.converter.ArtifactToSelectorEntity(ctx, a)
}

type pullRequestInfoConverter struct {
	converter PullRequestSelectorConverter
}

func newPullRequestInfoConverter(provider provifv1.Provider) *pullRequestInfoConverter {
	converter, err := provifv1.As[PullRequestSelectorConverter](provider)
	if err != nil {
		return nil
	}

	return &pullRequestInfoConverter{
		converter: converter,
	}
}

func (prc *pullRequestInfoConverter) toSelectorEntity(ctx context.Context, entity proto.Message) *internalpb.SelectorEntity {
	if prc == nil {
		return nil
	}

	p, ok := entity.(*minderv1.PullRequest)
	if !ok {
		return nil
	}

	return prc.converter.PullRequestToSelectorEntity(ctx, p)
}

// converterFactory is a map of entity types to their respective converters
type converterFactory struct {
	converters map[minderv1.Entity]entityInfoConverter
}

// newConverterFactory creates a new converterFactory with the default converters for each entity type
func newConverterFactory(provider provifv1.Provider) *converterFactory {
	return &converterFactory{
		converters: map[minderv1.Entity]entityInfoConverter{
			minderv1.Entity_ENTITY_REPOSITORIES:  newRepositoryInfoConverter(provider),
			minderv1.Entity_ENTITY_ARTIFACTS:     newArtifactInfoConverter(provider),
			minderv1.Entity_ENTITY_PULL_REQUESTS: newPullRequestInfoConverter(provider),
		},
	}
}

func (cf *converterFactory) getConverter(entityType minderv1.Entity) (entityInfoConverter, error) {
	conv, ok := cf.converters[entityType]
	if !ok {
		return nil, fmt.Errorf("no converter found for entity type %v", entityType)
	}

	return conv, nil
}

// EntityToSelectorEntity converts an entity to a SelectorEntity
func EntityToSelectorEntity(
	ctx context.Context,
	provider provifv1.Provider,
	entType minderv1.Entity,
	entity proto.Message,
) *internalpb.SelectorEntity {
	factory := newConverterFactory(provider)
	conv, err := factory.getConverter(entType)
	if err != nil {
		return nil
	}
	return conv.toSelectorEntity(ctx, entity)
}
