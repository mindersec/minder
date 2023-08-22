// Copyright 2023 Stacklok, Inc
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

package providers

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/oauth2"

	providerspb "github.com/stacklok/mediator/pkg/generated/protobuf/go/mediator/providers"
)

// RepositoryMetadata simplifies the import of both *Provider interfaces and
// types from the proto definition.
type RepositoryMetadata = providerspb.RepositoryMetadata

// RepoId copies providerspb.RepoId for simpler imports
type RepoId = providerspb.RepoId

type ArtifactId = providerspb.ArtifactId
type ArtifactVersionId = providerspb.ArtifactVersionId

// RepoProvider defines the methods which a provider needs to implement in
// order to manage repository status.
type RepoProvider interface {
	GetRepository(ctx context.Context, repo RepoId) (*RepositoryMetadata, error)
	GetBranchProtections(ctx context.Context, repo RepoId) ([]*providerspb.BranchProtectionPolicy, error)

	ListCallerRepositories(ctx context.Context, includeForks bool) ([]*RepositoryMetadata, error)
	ListRepositories(ctx context.Context, owner string) ([]*RepositoryMetadata, error)
}

type BuildProvider interface {
	GetBuildEnvironment(ctx context.Context, buildIdentifier string) (*BuildMetadata, error)

	ListCallerBuildEnvironments(ctx context.Context) ([]*BuildMetadata, error)
	ListBuildEnvironments(ctx context.Context, owner string) ([]*BuildMetadata, error)
}

type ArtifactProvider interface {
	GetArtifactVersions(ctx context.Context, artifact ArtifactId) ([]*providerspb.ArtifactVersion, error)
	GetArtifactVersion(ctx context.Context, version ArtifactVersionId)

	ListCallerArtifacts(ctx context.Context) ([]*ArtifactMetadata, error)
	ListArtifacts(ctx context.Context, owner string) ([]*ArtifactMetadata, error)
}

// ProviderFactory provides an interface for providers to create a new instance
// from the supplied credentials.  provider should match the name registered in
// RegisterProvider
type ProviderFactory func(ctx context.Context, provider string, token oauth2.Token) (RepoProvider, error)

var (
	providers    = make(map[string]ProviderFactory)
	providerLock = sync.Mutex{}
)

func RegisterProvider(name string, factory ProviderFactory) {
	providerLock.Lock()
	defer providerLock.Unlock()

	providers[name] = factory
}

func GetProvider(ctx context.Context, provider string) (RepoProvider, error) {
	providerLock.Lock()
	defer providerLock.Unlock()

	factory, ok := providers[provider]
	if !ok {
		return nil, fmt.Errorf("provider %q not registered", provider)
	}
	// TODO: extract token from context
	token, err := getAuthToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token from context: %w", err)
	}
	return factory(ctx, provider, *token)
}

var authTokenKey = struct{}{}

func WithAuthToken(ctx context.Context, token oauth2.Token) context.Context {
	return context.WithValue(ctx, authTokenKey, token)
}

func getAuthToken(ctx context.Context) (*oauth2.Token, error) {
	token, ok := ctx.Value(authTokenKey).(oauth2.Token)
	if !ok {
		return nil, fmt.Errorf("no auth token in context")
	}
	return &token, nil
}
