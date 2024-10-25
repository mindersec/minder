// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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

	"github.com/mindersec/minder/internal/crypto"
	"github.com/mindersec/minder/internal/providers/credentials"
	"github.com/mindersec/minder/internal/providers/github/clients"
	minderv1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
	"github.com/mindersec/minder/pkg/db"
	provinfv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// ErrProviderInvalidConfig is an error type which is returned when a provider configuration is invalid
type ErrProviderInvalidConfig struct {
	Details string
}

func (e ErrProviderInvalidConfig) Error() string {
	return fmt.Sprintf("invalid provider configuration: %s", e.Details)
}

// NewErrProviderInvalidConfig returns a new instance of ErrProviderInvalidConfig with details
// about the invalid configuration. This is meant for user-facing errors so that the only thing
// displayed to the user is the details of the error.
func NewErrProviderInvalidConfig(details string) ErrProviderInvalidConfig {
	return ErrProviderInvalidConfig{Details: details}
}

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
	case db.ProviderTypeImageLister:
		return minderv1.ProviderType_PROVIDER_TYPE_IMAGE_LISTER, true
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
) (string, error) {
	// if the provider doesn't support any auth flow
	// credentials state is not applicable
	if slices.Equal(prov.AuthFlows, []db.AuthorizationFlow{db.AuthorizationFlowNone}) {
		return provinfv1.CredentialStateNotApplicable, nil
	}

	cred, err := getCredentialForProvider(ctx, prov, cryptoEngine, s, provCfg)
	if err != nil {
		// One of the callers of this function treats this error as non-fatal
		// and uses the credState value.
		return provinfv1.CredentialStateUnset, err
	}

	// check if the credential is EmptyCredential
	// if it is, then the state is not applicable
	if _, ok := cred.(*credentials.EmptyCredential); ok {
		return provinfv1.CredentialStateUnset, nil
	}

	return provinfv1.CredentialStateSet, nil
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

	// TODO: get rid of this once we migrate all secrets to use the new structure
	var encryptedData crypto.EncryptedData
	if encToken.EncryptedAccessToken.Valid {
		encryptedData, err = crypto.DeserializeEncryptedData(encToken.EncryptedAccessToken.RawMessage)
		if err != nil {
			return nil, err
		}
	} else if encToken.EncryptedToken.Valid {
		encryptedData = crypto.NewBackwardsCompatibleEncryptedData(encToken.EncryptedToken.String)
	} else {
		return nil, fmt.Errorf("no secret found for provider %s", encToken.Provider)
	}
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
	_, cfg, err := clients.ParseAndMergeV1AppConfig(prov.Definition)
	if err != nil {
		return nil, fmt.Errorf("error parsing github app config: %w", err)
	}

	privateKey, err := provCfg.GitHubApp.GetPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("error reading private key: %w", err)
	}

	return credentials.NewGitHubInstallationTokenCredential(ctx, provCfg.GitHubApp.AppID, privateKey, cfg.GetEndpoint(),
		installation.AppInstallationID), nil
}
