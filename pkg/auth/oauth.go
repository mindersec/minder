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

// NOTE: This file is for stubbing out client code for proof of concept
// purposes. It will / should be removed in the future.
// Until then, it is not covered by unit tests and should not be used
// It does make a good example of how to use the generated client code
// for others to use as a reference.

package auth

import (
	"context"
	"fmt"
	"net/http"

	go_github "github.com/google/go-github/v53/github"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// Google OAuth2 provider
const Google = "google"

// Github OAuth2 provider
const Github = "github"

// NewOAuthConfig creates a new OAuth2 config for the given provider
// and whether the client is a CLI or web client
func NewOAuthConfig(provider string, cli bool) (*oauth2.Config, error) {
	redirectURL := func(provider string, cli bool) string {
		if cli {
			return fmt.Sprintf("http://localhost:8080/api/v1/auth/callback/%s/cli", provider)
		}
		return fmt.Sprintf("http://localhost:8080/api/v1/auth/callback/%s/web", provider)
	}

	scopes := func(provider string) []string {
		if provider == Google {
			return []string{"profile", "email"}
		}
		return []string{"user:email", "repo", "read:packages", "write:packages", "read:org"}
	}

	endpoint := func(provider string) oauth2.Endpoint {
		if provider == Google {
			return google.Endpoint
		}
		return github.Endpoint
	}

	if provider != Google && provider != Github {
		return nil, fmt.Errorf("invalid provider: %s", provider)
	}

	return &oauth2.Config{
		ClientID:     viper.GetString(fmt.Sprintf("%s.client_id", provider)),
		ClientSecret: viper.GetString(fmt.Sprintf("%s.client_secret", provider)),
		RedirectURL:  redirectURL(provider, cli),
		Scopes:       scopes(provider),
		Endpoint:     endpoint(provider),
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
func ValidateProviderToken(_ context.Context, provider string, token string) error {
	if provider == Github {
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
