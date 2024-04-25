//
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

// Package keycloak provides an implementation of the Keycloak IdentityProvider.
package keycloak

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/stacklok/minder/internal/auth"
	"github.com/stacklok/minder/internal/auth/keycloak/client"
	serverconfig "github.com/stacklok/minder/internal/config/server"
	"github.com/stacklok/minder/internal/util/ptr"
)

// KeyCloak is an implementation of the auth.IdentityProvider interface.
type KeyCloak struct {
	name string
	url  url.URL
	cfg  serverconfig.IdentityConfig

	kcClient client.ClientWithResponsesInterface
}

// NewKeyCloak creates a new KeyCloak identity provider.
func NewKeyCloak(name string, cfg serverconfig.IdentityConfig) (*KeyCloak, error) {
	kcUrl := cfg.Issuer()
	httpClient, err := newAuthorizedClient(cfg.Issuer(), cfg)
	if err != nil {
		return nil, err
	}
	kcClient, err := client.NewClientWithResponses(kcUrl.String(), client.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}

	return &KeyCloak{
		name:     name,
		url:      cfg.Issuer(),
		cfg:      cfg,
		kcClient: kcClient,
	}, nil
}

var _ auth.IdentityProvider = (*KeyCloak)(nil)

var errNotFound = errors.New("user not found in identity store")

// String implements auth.IdentityProvider.
func (k *KeyCloak) String() string {
	return k.name
}

// URL implements auth.IdentityProvider.
func (k *KeyCloak) URL() url.URL {
	return k.url
}

// Resolve implements auth.IdentityProvider.
func (k *KeyCloak) Resolve(ctx context.Context, id string) (*auth.Identity, error) {
	remoteUser, err := k.lookupUser(ctx, id)
	if err != nil {
		// TODO: pass through to the next statements to try to create the user if not existing
		return nil, fmt.Errorf("unable to resolve user: %w", err)
	}
	if remoteUser != nil {
		return remoteUser, nil
	}

	// The user doesn't already existing in Keycloak, try to create a user
	// based on GitHub metadata.  When the user logs in, they should end up
	// using the pre-created Keycloak identity.
	ghUser, err := k.lookupGithubUser(ctx, id)
	if err != nil {
		// TODO: how to signal GH user does not exist separate from lookup error?
		return nil, fmt.Errorf("unable to resolve github user: %w", err)
	}

	return k.createKeycloakUser(ctx, ghUser)
}

// Validate implements auth.IdentityProvider.
func (k *KeyCloak) Validate(_ context.Context, token jwt.Token) (*auth.Identity, error) {
	// TODO: implement validating the JWT against the jwks.

	humanName, ok := token.Get("preferred_username")
	if !ok {
		return nil, errors.New("preferred_username not found in token")
	}
	humanStr, ok := humanName.(string)
	if !ok {
		return nil, errors.New("preferred_username is not a string")
	}
	return &auth.Identity{
		UserID:    token.Subject(),
		HumanName: humanStr,
		Provider:  k,
	}, nil
}

func (k *KeyCloak) lookupUser(ctx context.Context, id string) (*auth.Identity, error) {
	// First, look up by user ID
	resp, err := k.kcClient.GetAdminRealmsRealmUsersUserIdWithResponse(ctx, "stacklok", id, nil)
	if err == nil && resp.StatusCode() == http.StatusOK {
		id := k.userToIdentity(*resp.JSON200)
		if id != nil {
			return id, nil
		}
	}

	// next, try lookup by GitHub login
	userLookup, err := k.kcClient.GetAdminRealmsRealmUsersWithResponse(ctx, "stacklok", &client.GetAdminRealmsRealmUsersParams{
		Exact:    ptr.Ptr(true),
		Username: &id,
	})
	if err == nil && userLookup.StatusCode() == http.StatusOK && len(*userLookup.JSON200) == 1 {
		id := k.userToIdentity((*userLookup.JSON200)[0])
		if id != nil {
			return id, nil
		}
	}

	// last, try lookup by GitHub numeric ID
	userLookup, err = k.kcClient.GetAdminRealmsRealmUsersWithResponse(ctx, "stacklok", &client.GetAdminRealmsRealmUsersParams{
		Q: ptr.Ptr(fmt.Sprintf("gh_id:%s", id)),
	})
	if err == nil && userLookup.StatusCode() == http.StatusOK && len(*userLookup.JSON200) == 1 {
		id := k.userToIdentity((*userLookup.JSON200)[0])
		if id != nil {
			return id, nil
		}
	}

	return nil, errNotFound
}

func (k *KeyCloak) userToIdentity(user client.UserRepresentation) *auth.Identity {
	if user.Attributes == nil || user.Id == nil {
		return nil
	}
	return &auth.Identity{
		UserID:    *user.Id,
		HumanName: *user.Username,
		Provider:  k,
	}
}

func (_ *KeyCloak) lookupGithubUser(_ context.Context, _ string) (*auth.Identity, error) {
	return nil, fmt.Errorf("not implemented")
}

func (_ *KeyCloak) createKeycloakUser(_ context.Context, _ *auth.Identity) (*auth.Identity, error) {
	return nil, fmt.Errorf("not implemented")
}

func newAuthorizedClient(kcUrl url.URL, cfg serverconfig.IdentityConfig) (*http.Client, error) {
	tokenUrl := kcUrl.JoinPath("realms/stacklok/protocol/openid-connect/token")
	clientSecret, err := cfg.GetClientSecret()
	if err != nil {
		return nil, err
	}
	clientCredentials := clientcredentials.Config{
		ClientID:     cfg.ClientId,
		ClientSecret: clientSecret,
		TokenURL:     tokenUrl.String(),
	}

	// verify that we can fetch a token before returning the client
	if _, err = clientCredentials.Token(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	return clientCredentials.Client(context.Background()), nil
}
