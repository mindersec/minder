// Copyright 2023 Stacklok, Inc.
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

// Package providers contains general utilities for interacting with
// providers.
package providers

import (
	"context"
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/stacklok/mediator/internal/crypto"
	"github.com/stacklok/mediator/internal/db"
	gitclient "github.com/stacklok/mediator/internal/providers/git"
	ghclient "github.com/stacklok/mediator/internal/providers/github"
	httpclient "github.com/stacklok/mediator/internal/providers/http"
	provinfv1 "github.com/stacklok/mediator/pkg/providers/v1"
)

// GetProviderBuilder is a utility function which allows for the creation of
// a provider factory.
func GetProviderBuilder(
	ctx context.Context,
	prov db.Provider,
	groupID int32,
	store db.Store,
	crypteng *crypto.Engine,
) (*ProviderBuilder, error) {
	encToken, err := store.GetAccessTokenByGroupID(ctx,
		db.GetAccessTokenByGroupIDParams{Provider: prov.Name, GroupID: groupID})
	if err != nil {
		return nil, fmt.Errorf("error getting access token: %w", err)
	}

	decryptedToken, err := crypteng.DecryptOAuthToken(encToken.EncryptedToken)
	if err != nil {
		return nil, fmt.Errorf("error decrypting access token: %w", err)
	}

	return NewProviderBuilder(&prov, encToken, decryptedToken.AccessToken), nil
}

// ProviderBuilder is a utility struct which allows for the creation of
// provider clients.
type ProviderBuilder struct {
	p        *db.Provider
	tokenInf db.ProviderAccessToken
	tok      string
}

// NewProviderBuilder creates a new provider builder.
func NewProviderBuilder(
	p *db.Provider,
	tokenInf db.ProviderAccessToken,
	tok string,
) *ProviderBuilder {
	return &ProviderBuilder{
		p:        p,
		tokenInf: tokenInf,
		tok:      tok,
	}
}

// Implements returns true if the provider implements the given type.
func (pb *ProviderBuilder) Implements(impl db.ProviderType) bool {
	return slices.Contains(pb.p.Implements, impl)
}

// GetName returns the name of the provider instance as defined in the
// database.
func (pb *ProviderBuilder) GetName() string {
	return pb.p.Name
}

// GetToken returns the token for the provider.
func (pb *ProviderBuilder) GetToken() string {
	return pb.tok
}

// GetGit returns a git client for the provider.
func (pb *ProviderBuilder) GetGit() (*gitclient.Git, error) {
	if !pb.Implements(db.ProviderTypeGit) {
		return nil, fmt.Errorf("provider does not implement git")
	}

	return gitclient.NewGit(pb.tok), nil
}

// GetHTTP returns a github client for the provider.
func (pb *ProviderBuilder) GetHTTP(ctx context.Context) (provinfv1.REST, error) {
	if !pb.Implements(db.ProviderTypeRest) {
		return nil, fmt.Errorf("provider does not implement rest")
	}

	// We can re-use the GitHub provider in case it also implements GitHub.
	// The client gives us the ability to handle rate limiting and other
	// things.
	if pb.Implements(db.ProviderTypeGithub) {
		return pb.GetGitHub(ctx)
	}

	if pb.p.Version != provinfv1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	// TODO: Parsing will change based on version
	cfg, err := httpclient.ParseV1Config(pb.p.Definition)
	if err != nil {
		return nil, fmt.Errorf("error parsing http config: %w", err)
	}

	return httpclient.NewREST(cfg, pb.tok)
}

// GetGitHub returns a github client for the provider.
func (pb *ProviderBuilder) GetGitHub(ctx context.Context) (*ghclient.RestClient, error) {
	if !pb.Implements(db.ProviderTypeGithub) {
		return nil, fmt.Errorf("provider does not implement github")
	}

	if pb.p.Version != provinfv1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	// TODO: Parsing will change based on version
	cfg, err := ghclient.ParseV1Config(pb.p.Definition)
	if err != nil {
		return nil, fmt.Errorf("error parsing github config: %w", err)
	}

	cli, err := ghclient.NewRestClient(ctx, cfg, pb.GetToken(), pb.tokenInf.OwnerFilter.String)
	if err != nil {
		return nil, fmt.Errorf("error creating github client: %w", err)
	}

	return cli, nil
}
