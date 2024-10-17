// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
	"net/http"

	go_github "github.com/google/go-github/v63/github"
	"github.com/spf13/viper"
)

const (
	// Github OAuth2 provider
	Github = "github"
)

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
