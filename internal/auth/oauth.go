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
	"os"
	"path/filepath"
	"slices"

	go_github "github.com/google/go-github/v60/github"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/stacklok/minder/internal/config"
)

const (
	// Google OAuth2 provider
	Google = "google"

	// Github OAuth2 provider
	Github = "github"

	// GitHubApp provider
	GitHubApp = "github-app"
)

// TODO:
var knownProviders = []string{Google, Github, GitHubApp}

// NewOAuthConfig creates a new OAuth2 config for the given provider
// and whether the client is a CLI or web client
func NewOAuthConfig(provider string, cli bool) (*oauth2.Config, error) {
	redirectURL := func(provider string, cli bool) string {
		base := viper.GetString(fmt.Sprintf("%s.redirect_uri", provider))
		if cli {
			return fmt.Sprintf("%s/cli", base)
		}
		return fmt.Sprintf("%s/web", base)
	}

	scopes := func(provider string) []string {
		if provider == Google {
			return []string{"profile", "email"}
		}
		if provider == GitHubApp {
			return []string{}
		}
		return []string{"user:email", "repo", "read:packages", "write:packages", "workflow", "read:org"}
	}

	endpoint := func(provider string) oauth2.Endpoint {
		if provider == Google {
			return google.Endpoint
		}
		return github.Endpoint
	}

	if !slices.Contains(knownProviders, provider) {
		return nil, fmt.Errorf("invalid provider: %s", provider)
	}

	clientId, err := readFileOrConfig(fmt.Sprintf("%s.client_id", provider))
	if err != nil {
		return nil, fmt.Errorf("failed to read %s.client_id: %w", provider, err)
	}
	clientSecret, err := readFileOrConfig(fmt.Sprintf("%s.client_secret", provider))
	if err != nil {
		return nil, fmt.Errorf("failed to read %s.client_id: %w", provider, err)
	}

	return &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL(provider, cli),
		Scopes:       scopes(provider),
		Endpoint:     endpoint(provider),
	}, nil
}

// readFileOrConfig prefers reading from configKey_file (for Kubernetes distribution
// of secrets), but falls back to a viper string value if the file is not present.
func readFileOrConfig(configKey string) (string, error) {
	fileKey := configKey + "_file"
	if viper.IsSet(fileKey) && viper.GetString(fileKey) != "" {
		filename := viper.GetString(fileKey)
		// filepath.Clean avoids a gosec warning on reading a file by name
		data, err := os.ReadFile(filepath.Clean(filename))
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return viper.GetString(configKey), nil
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

// RegisterOAuthFlags registers client ID and secret file flags for all known
// providers.  This is pretty tied into the internal of the auth module, so it
// lives here, but it would be nice if we have a consistent registration
// pattern (database flags are registered in the config module).
func RegisterOAuthFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	for _, provider := range knownProviders {
		idFileKey := fmt.Sprintf("%s.client_id_file", provider)
		idFileFlag := fmt.Sprintf("%s-client-id-file", provider)
		idFileDesc := fmt.Sprintf("File containing %s client ID", provider)
		secretFileKey := fmt.Sprintf("%s.client_secret_file", provider)
		secretFileFlag := fmt.Sprintf("%s-client-secret-file", provider)
		secretFileDesc := fmt.Sprintf("File containing %s client secret", provider)
		if err := config.BindConfigFlag(
			v, flags, idFileKey, idFileFlag, "", idFileDesc, flags.String); err != nil {
			return err
		}
		if err := config.BindConfigFlag(
			v, flags, secretFileKey, secretFileFlag, "", secretFileDesc, flags.String); err != nil {
			return err
		}
	}
	return nil
}
