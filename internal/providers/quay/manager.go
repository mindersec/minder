// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package quay

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers/credentials"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	v1 "github.com/mindersec/minder/pkg/providers/v1"
)

type providerClassManager struct {
	store    db.Store
	crypteng crypto.Engine
}

// NewQuayProviderClassManager creates a new provider class manager for the quay provider
func NewQuayProviderClassManager(crypteng crypto.Engine, store db.Store) *providerClassManager {
	return &providerClassManager{
		store:    store,
		crypteng: crypteng,
	}
}

// GetSupportedClasses implements the ProviderClassManager interface
func (*providerClassManager) GetSupportedClasses() []db.ProviderClass {
	return []db.ProviderClass{db.ProviderClassQuay}
}

func (g *providerClassManager) GetProviderClassInfo(class db.ProviderClass) (*minderv1.ProviderClassInfo, error) {
	if !slices.Contains(g.GetSupportedClasses(), class) {
		return nil, fmt.Errorf("provider does not implement %s", class)
	}

	return ClassInfo(), nil
}

// Build implements the ProviderClassManager interface
func (g *providerClassManager) Build(ctx context.Context, config *db.Provider) (v1.Provider, error) {
	class := config.Class
	// This should be validated by the caller, but let's check anyway
	if !slices.Contains(g.GetSupportedClasses(), class) {
		return nil, fmt.Errorf("provider does not implement quay")
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
		return nil, fmt.Errorf("error parsing quay provider config: %w", err)
	}

	cli, err := New(
		creds,
		cfg,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating quay client: %w", err)
	}
	return cli, nil
}

// Delete implements the ProviderClassManager interface
// TODO: Implement this
func (*providerClassManager) Delete(_ context.Context, _ *db.Provider) error {
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

	if !encToken.EncryptedAccessToken.Valid {
		return nil, fmt.Errorf("no secret found for provider %s", encToken.Provider)
	}
	encryptedData, err := crypto.DeserializeEncryptedData(encToken.EncryptedAccessToken.RawMessage)
	if err != nil {
		return nil, err
	}
	decryptedToken, err := m.crypteng.DecryptOAuthToken(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("error decrypting access token: %w", err)
	}

	return credentials.NewOAuth2TokenCredential(decryptedToken.AccessToken), nil
}

func (m *providerClassManager) MarshallConfig(
	_ context.Context, class db.ProviderClass, config json.RawMessage,
) (json.RawMessage, error) {
	if !slices.Contains(m.GetSupportedClasses(), class) {
		return nil, fmt.Errorf("provider does not implement %s", string(class))
	}

	return MarshalV1Config(config)
}
