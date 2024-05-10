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
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/exp/slices"

	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/crypto"
	"github.com/stacklok/minder/internal/db"
	"github.com/stacklok/minder/internal/providers/credentials"
	"github.com/stacklok/minder/internal/providers/github/clients"
	minderv1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
	provinfv1 "github.com/stacklok/minder/pkg/providers/v1"
)

// DBToPBType converts a database provider type to a protobuf provider type.
func DBToPBType(t db.ProviderType) (minderv1.ProviderType, bool) {
	switch t {
	case db.ProviderTypeGit:
		return minderv1.ProviderType_PROVIDER_TYPE_GIT, true
	case db.ProviderTypeGithub:
		return minderv1.ProviderType_PROVIDER_TYPE_GITHUB, true
	case db.ProviderTypeRest:
		return minderv1.ProviderType_PROVIDER_TYPE_REST, true
	case db.ProviderTypeRepoLister:
		return minderv1.ProviderType_PROVIDER_TYPE_REPO_LISTER, true
	case db.ProviderTypeOci:
		return minderv1.ProviderType_PROVIDER_TYPE_OCI, true
	default:
		return minderv1.ProviderType_PROVIDER_TYPE_UNSPECIFIED, false
	}
}

// DBToPBAuthFlow converts a database authorization flow to a protobuf authorization flow.
func DBToPBAuthFlow(t db.AuthorizationFlow) (minderv1.AuthorizationFlow, bool) {
	switch t {
	case db.AuthorizationFlowNone:
		return minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_NONE, true
	case db.AuthorizationFlowUserInput:
		return minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_USER_INPUT, true
	case db.AuthorizationFlowOauth2AuthorizationCodeFlow:
		return minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_OAUTH2_AUTHORIZATION_CODE_FLOW, true
	case db.AuthorizationFlowGithubAppFlow:
		return minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_GITHUB_APP_FLOW, true
	default:
		return minderv1.AuthorizationFlow_AUTHORIZATION_FLOW_UNSPECIFIED, false
	}
}

// GetCredentialStateForProvider returns the credential state for the given provider.
func GetCredentialStateForProvider(
	ctx context.Context,
	prov db.Provider,
	s db.Store,
	cryptoEngine crypto.Engine,
	provCfg *serverconfig.ProviderConfig,
) string {
	var credState string
	// if the provider doesn't support any auth flow
	// credentials state is not applicable
	if slices.Equal(prov.AuthFlows, []db.AuthorizationFlow{db.AuthorizationFlowNone}) {
		credState = provinfv1.CredentialStateNotApplicable
	} else {
		credState = provinfv1.CredentialStateUnset
		cred, err := getCredentialForProvider(ctx, prov, cryptoEngine, s, provCfg)
		if err != nil {
			// This is non-fatal
			zerolog.Ctx(ctx).Error().Err(err).Str("provider", prov.Name).Msg("error getting credential")
		} else {
			// check if the credential is EmptyCredential
			// if it is, then the state is not applicable
			if _, ok := cred.(*credentials.EmptyCredential); ok {
				credState = provinfv1.CredentialStateUnset
			} else {
				credState = provinfv1.CredentialStateSet
			}
		}
	}

	return credState
}

func getCredentialForProvider(
	ctx context.Context,
	prov db.Provider,
	crypteng crypto.Engine,
	store db.Store,
	cfg *serverconfig.ProviderConfig,
) (provinfv1.Credential, error) {
	encToken, err := store.GetAccessTokenByProjectID(ctx,
		db.GetAccessTokenByProjectIDParams{Provider: prov.Name, ProjectID: prov.ProjectID})
	if errors.Is(err, sql.ErrNoRows) {
		zerolog.Ctx(ctx).Debug().Msg("no access token found for provider")

		// If we don't have an access token, check if we have an installation ID
		return getInstallationTokenCredential(ctx, prov, store, cfg)
	} else if err != nil {
		return nil, fmt.Errorf("error getting credential: %w", err)
	}

	// TODO: get rid of this once we store the EncryptedData struct in
	// the database.
	encryptedData := crypto.NewBackwardsCompatibleEncryptedData(encToken.EncryptedToken)
	decryptedToken, err := crypteng.DecryptOAuthToken(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("error decrypting access token: %w", err)
	}
	zerolog.Ctx(ctx).Debug().Msg("access token found for provider")
	return credentials.NewGitHubTokenCredential(decryptedToken.AccessToken), nil
}

// getInstallationTokenCredential returns a GitHub installation token credential if the provider has an installation ID
func getInstallationTokenCredential(
	ctx context.Context,
	prov db.Provider,
	store db.Store,
	provCfg *serverconfig.ProviderConfig,
) (provinfv1.Credential, error) {
	installation, err := store.GetInstallationIDByProviderID(ctx, uuid.NullUUID{
		UUID:  prov.ID,
		Valid: true,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return credentials.NewEmptyCredential(), nil
	} else if err != nil {
		return nil, fmt.Errorf("error getting installation ID: %w", err)
	}
	cfg, err := clients.ParseV1AppConfig(prov.Definition)
	if err != nil {
		return nil, fmt.Errorf("error parsing github app config: %w", err)
	}

	privateKey, err := provCfg.GitHubApp.GetPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("error reading private key: %w", err)
	}

	return credentials.NewGitHubInstallationTokenCredential(ctx, provCfg.GitHubApp.AppID, privateKey, cfg.Endpoint,
		installation.AppInstallationID), nil
}
