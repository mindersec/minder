// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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

	"github.com/mindersec/minder/internal/auth"
	"github.com/mindersec/minder/internal/auth/keycloak/client"
	"github.com/mindersec/minder/internal/util/ptr"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
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
	return nil, errNotFound
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
	ret := &auth.Identity{
		UserID:    *user.Id,
		HumanName: *user.Username,
		Provider:  k,
	}
	// If the user has a first and last name, return them too
	if user.FirstName != nil {
		ret.FirstName = *user.FirstName
	}
	if user.LastName != nil {
		ret.LastName = *user.LastName
	}
	return ret
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
