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
	"fmt"

	"github.com/stacklok/mediator/internal/crypto"
	"github.com/stacklok/mediator/internal/db"
	ghclient "github.com/stacklok/mediator/internal/providers/github"
)

// BuildClient is a utility function which allows for building
// a client for a provider from a groupID and provider name.
//
// TODO: This should return an abstract provider interface
// instead of a concrete github client.
func BuildClient(
	ctx context.Context,
	prov string,
	groupID int32,
	store db.Store,
	crypteng *crypto.Engine,
) (ghclient.RestAPI, error) {
	encToken, err := store.GetAccessTokenByGroupID(ctx,
		db.GetAccessTokenByGroupIDParams{Provider: prov, GroupID: groupID})
	if err != nil {
		return nil, fmt.Errorf("error getting access token: %w", err)
	}

	decryptedToken, err := crypteng.DecryptOAuthToken(encToken.EncryptedToken)
	if err != nil {
		return nil, fmt.Errorf("error decrypting access token: %w", err)
	}

	cli, err := ghclient.NewRestClient(ctx, ghclient.GitHubConfig{
		Token: decryptedToken.AccessToken,
	}, encToken.OwnerFilter.String)
	if err != nil {
		return nil, fmt.Errorf("error creating github client: %w", err)
	}

	return cli, nil
}
