//
// Copyright 2023 Stacklok, Inc.
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

package auth

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	go_github "github.com/google/go-github/v61/github"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	"github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/db"
)

const (
	// Github OAuth2 provider
	Github = "github"

	// GitHubApp provider
	GitHubApp = "github-app"
)

var knownProviders = []string{Github, GitHubApp}

func getOAuthClientConfig(c *server.ProviderConfig, provider string) (*server.OAuthClientConfig, error) {
	var oc *server.OAuthClientConfig
	var err error

	if !slices.Contains(knownProviders, provider) {
		return nil, fmt.Errorf("invalid provider: %s", provider)
	}

	// first read the new provider-nested key. If it's missing, fallback to using the older
	// top-level keys.
	switch provider {
	case Github:
		if c != nil && c.GitHub != nil {
			oc = &c.GitHub.OAuthClientConfig
		}
		fallbackOAuthClientConfigValues("github", oc)
	case GitHubApp:
		if c != nil && c.GitHubApp != nil {
			oc = &c.GitHubApp.OAuthClientConfig
		}
		fallbackOAuthClientConfigValues("github-app", oc)
	default:
		err = fmt.Errorf("unknown provider: %s", provider)
	}

	return oc, err
}

func fallbackOAuthClientConfigValues(provider string, cfg *server.OAuthClientConfig) {
	// we read the values one-by-one instead of just getting the top-level key to allow
	// for environment variables to be set per-variable
	cfg.ClientID = viper.GetString(fmt.Sprintf("%s.client_id", provider))
	cfg.ClientIDFile = viper.GetString(fmt.Sprintf("%s.client_id_file", provider))
	cfg.ClientSecret = viper.GetString(fmt.Sprintf("%s.client_secret", provider))
	cfg.ClientSecretFile = viper.GetString(fmt.Sprintf("%s.client_secret_file", provider))
	cfg.RedirectURI = viper.GetString(fmt.Sprintf("%s.redirect_uri", provider))
}

// NewOAuthConfig creates a new OAuth2 config for the given provider
// and whether the client is a CLI or web client
func NewOAuthConfig(c *server.ProviderConfig, provider string, cli bool) (*oauth2.Config, error) {
	oauthConfig, err := getOAuthClientConfig(c, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth client config: %w", err)
	}

	redirectURL := func(provider string, cli bool) string {
		base := oauthConfig.RedirectURI
		if provider == GitHubApp {
			// GitHub App does not distinguish between CLI and web clients
			return base
		}
		if cli {
			return fmt.Sprintf("%s/cli", base)
		}
		return fmt.Sprintf("%s/web", base)
	}

	scopes := func(provider string) []string {
		if provider == GitHubApp {
			return []string{}
		}
		return []string{"user:email", "repo", "read:packages", "write:packages", "workflow", "read:org"}
	}

	endpoint := func() oauth2.Endpoint {
		return github.Endpoint
	}

	clientID, err := oauthConfig.GetClientID()
	if err != nil {
		return nil, fmt.Errorf("failed to get client ID: %w", err)
	}

	clientSecret, err := oauthConfig.GetClientSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to get client secret: %w", err)
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL(provider, cli),
		Scopes:       scopes(provider),
		Endpoint:     endpoint(),
	}, nil
}

// NewProviderHttpClient creates a new http client for the given provider
func NewProviderHttpClient(provider string) *http.Client {
	if provider == Github {
		hClient := &http.Client{
			Transport: &go_github.BasicAuthTransport{
				Username: viper.GetString(fmt.Sprintf("%s.client_id", provider)),
				Password: viper.GetString(fmt.Sprintf("%s.client_secret", provider)),
			},
		}
		return hClient
	}
	return nil
}

// DeleteAccessToken deletes the access token for a given provider
func DeleteAccessToken(ctx context.Context, provider string, token string) error {
	hClient := NewProviderHttpClient(provider)
	if hClient == nil {
		return fmt.Errorf("invalid provider: %s", provider)
	}

	client := go_github.NewClient(hClient)
	client_id := viper.GetString(fmt.Sprintf("%s.client_id", provider))
	_, err := client.Authorizations.Revoke(ctx, client_id, token)

	if err != nil {
		return err
	}
	return nil
}

// ValidateProviderToken validates the given token for the given provider
func ValidateProviderToken(_ context.Context, provider db.ProviderClass, token string) error {
	// Fixme: this should really be handled by the provider. Should this be in the credentials API or the manager?
	if provider == db.ProviderClassGithub {
		// Create an OAuth2 token source with the PAT
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})

		// Create an authenticated GitHub client
		oauth2Client := oauth2.NewClient(context.Background(), tokenSource)
		client := go_github.NewClient(oauth2Client)

		// Make a sample API request to check token validity
		_, _, err := client.Users.Get(context.Background(), "")
		if err != nil {
			return fmt.Errorf("invalid token: %s", err)
		}
		return nil
	}
	return fmt.Errorf("invalid provider: %s", provider)
}
