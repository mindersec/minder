// Copyright 2024 Stacklok, Inc
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

package dockerhub

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/credentials"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

type providerClassManager struct {
	store    db.Store
	crypteng crypto.Engine
}

// NewDockerHubProviderClassManager creates a new provider class manager for the dockerhub provider
func NewDockerHubProviderClassManager(crypteng crypto.Engine, store db.Store) *providerClassManager {
	return &providerClassManager{
		store:    store,
		crypteng: crypteng,
	}
}

// GetSupportedClasses implements the ProviderClassManager interface
func (_ *providerClassManager) GetSupportedClasses() []db.ProviderClass {
	return []db.ProviderClass{db.ProviderClassDockerhub}
}

// Build implements the ProviderClassManager interface
func (g *providerClassManager) Build(ctx context.Context, config *db.Provider) (v1.Provider, error) {
	class := config.Class
	// This should be validated by the caller, but let's check anyway
	if !slices.Contains(g.GetSupportedClasses(), class) {
		return nil, fmt.Errorf("provider does not implement github")
	}

	if config.Version != v1.V1 {
		return nil, fmt.Errorf("provider version not supported")
	}

	creds, err := g.getProviderCredentials(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch credentials")
	}

	cfg, err := ParseV1Config(config.Definition)
	if err != nil {
		return nil, fmt.Errorf("error parsing github app config: %w", err)
	}

	cli, err := New(
		creds,
		cfg,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating github app client: %w", err)
	}
	return cli, nil
}

// Delete implements the ProviderClassManager interface
// TODO: Implement this
func (_ *providerClassManager) Delete(_ context.Context, _ *db.Provider) error {
	return nil
}

func (m *providerClassManager) getProviderCredentials(
	ctx context.Context,
	prov *db.Provider,
) (v1.OAuth2TokenCredential, error) {
	encToken, err := m.store.GetAccessTokenByProjectID(ctx,
		db.GetAccessTokenByProjectIDParams{Provider: prov.Name, ProjectID: prov.ProjectID})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("error getting credential: %w", err)
	}

	decryptedToken, err := m.crypteng.DecryptOAuthToken(encToken.EncryptedToken)
	if err != nil {
		return nil, fmt.Errorf("error decrypting access token: %w", err)
	}

	return credentials.NewOAuth2TokenCredential(decryptedToken.AccessToken), nil
}
