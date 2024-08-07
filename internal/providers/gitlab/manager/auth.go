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

package manager

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/gitlab"

	"github.com/stacklok/minder/internal/db"
	m "github.com/stacklok/minder/internal/providers/manager"
	provv1 "github.com/stacklok/minder/pkg/providers/v1"
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
	if oauthClientConfig.Endpoint != nil {
		oauthConfig.Endpoint = oauth2.Endpoint{
			TokenURL: oauthClientConfig.Endpoint.TokenURL,
		}
	}

	oauthConfig.ClientID = clientId
	oauthConfig.ClientSecret = clientSecret
	return oauthConfig, nil
}

// ValidateCredentials implements the providerClassOAuthManager interface
func (_ *providerClassManager) ValidateCredentials(
	_ context.Context, cred provv1.Credential, _ *m.CredentialVerifyParams,
) error {
	tokenCred, ok := cred.(provv1.OAuth2TokenCredential)
	if !ok {
		return fmt.Errorf("invalid credential type: %T", cred)
	}

	_, err := tokenCred.GetAsOAuth2TokenSource().Token()
	if err != nil {
		return fmt.Errorf("cannot get token from credential: %w", err)
	}

	// TODO: verify token identity
	// if params.RemoteUser != "" {
	// 	err := g.ghService.VerifyProviderTokenIdentity(ctx, params.RemoteUser, token.AccessToken)
	// 	if err != nil {
	// 		return fmt.Errorf("error verifying token identity: %w", err)
	// 	}
	// } else {
	// 	zerolog.Ctx(ctx).Warn().Msg("RemoteUser not found in session state")
	// }

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
