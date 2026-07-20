// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/gitlab"

	"github.com/mindersec/minder/internal/db"
	m "github.com/mindersec/minder/internal/providers/manager"
	provv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// NewOAuthConfig implements the providerClassOAuthManager interface
func (g *providerClassManager) NewOAuthConfig(_ db.ProviderClass, cli bool) (*oauth2.Config, error) {
	oauthClientConfig := &g.glpcfg.OAuthClientConfig
	oauthConfig := getOauthConfig(oauthClientConfig.RedirectURI, cli, g.glpcfg.Scopes)

	clientId, err := oauthClientConfig.GetClientID()
	if err != nil {
		return nil, fmt.Errorf("failed to get client ID: %w", err)
	}

	clientSecret, err := oauthClientConfig.GetClientSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to get client secret: %w", err)
	}

	// this is currently only used for testing as github uses well-known endpoints
	if oauthClientConfig.Endpoint != nil && oauthClientConfig.Endpoint.TokenURL != "" {
		oauthConfig.Endpoint = oauth2.Endpoint{
			TokenURL: oauthClientConfig.Endpoint.TokenURL,
		}
	}

	oauthConfig.ClientID = clientId
	oauthConfig.ClientSecret = clientSecret
	return oauthConfig, nil
}

// ValidateCredentials implements the providerClassOAuthManager interface
func (*providerClassManager) ValidateCredentials(
	_ context.Context, cred provv1.Credential, _ *m.CredentialVerifyParams,
) error {
	switch c := cred.(type) {
	case provv1.OAuth2TokenCredential:
		_, err := c.GetAsOAuth2TokenSource().Token()
		if err != nil {
			return fmt.Errorf("cannot get token from credential: %w", err)
		}
	case string:
		if !strings.HasPrefix(c, "glpat-") {
			return fmt.Errorf("expected token to start with 'glpat-'")
		}
	default:
		return fmt.Errorf("invalid credential type: %T", cred)
	}

	return nil
}

func getOauthConfig(redirectUrlBase string, cli bool, scopes []string) *oauth2.Config {
	var redirectUrl string

	if cli {
		redirectUrl = fmt.Sprintf("%s/cli", redirectUrlBase)
	} else {
		redirectUrl = fmt.Sprintf("%s/web", redirectUrlBase)
	}

	return &oauth2.Config{
		RedirectURL: redirectUrl,
		Scopes:      scopes,
		// TODO: This should come from the provider config
		Endpoint: gitlab.Endpoint,
	}
}
