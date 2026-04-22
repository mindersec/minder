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
	name  string
	url   url.URL
	realm string
	cfg   serverconfig.IdentityConfig

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

	oidcCfg, err := cfg.DiscoverOIDCEndpoints(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC endpoints: %w", err)
	}

	parsedIssuerUrl, err := url.Parse(oidcCfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to parse discovered issuer: %w", err)
	}

	return &KeyCloak{
		name:     name,
		url:      *parsedIssuerUrl,
		realm:    cfg.Realm,
		cfg:      cfg,
		kcClient: kcClient,
	}, nil
}

var _ auth.IdentityManager = (*KeyCloak)(nil)
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
	// Note: Currently, JWTs are validated before this method is called within internal/auth/jwt/validator.go.

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

// DeleteUser deletes a user from Keycloak
func (k *KeyCloak) DeleteUser(ctx context.Context, userID string) error {
	resp, err := k.kcClient.DeleteAdminRealmsRealmUsersUserIdWithResponse(ctx, k.realm, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	if resp.StatusCode() != http.StatusNoContent && resp.StatusCode() != http.StatusNotFound {
		return fmt.Errorf("unexpected status code when deleting user: %d", resp.StatusCode())
	}
	return nil
}

// GetEvents returns account events from Keycloak
func (k *KeyCloak) GetEvents(ctx context.Context) ([]auth.AccountEvent, error) {
	resp, err := k.kcClient.GetAdminRealmsRealmEventsWithResponse(ctx, k.realm, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code fetching events: %d", resp.StatusCode())
	}

	var events []auth.AccountEvent
	for _, e := range *resp.JSON200 {
		events = append(events, auth.AccountEvent{
			Time:   ptr.ValueOrZero(e.Time),
			Type:   ptr.ValueOrZero(e.Type),
			UserId: ptr.ValueOrZero(e.UserId),
		})
	}
	return events, nil
}

// GetAdminEvents returns administrative events from Keycloak
func (k *KeyCloak) GetAdminEvents(ctx context.Context, operationTypes, resourceTypes []string) ([]auth.AdminEvent, error) {
	params := &client.GetAdminRealmsRealmAdminEventsParams{
		OperationTypes: &operationTypes,
		ResourceTypes:  &resourceTypes,
	}
	resp, err := k.kcClient.GetAdminRealmsRealmAdminEventsWithResponse(ctx, k.realm, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin events: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code fetching admin events: %d", resp.StatusCode())
	}

	var events []auth.AdminEvent
	for _, e := range *resp.JSON200 {
		events = append(events, auth.AdminEvent{
			Time:          ptr.ValueOrZero(e.Time),
			OperationType: ptr.ValueOrZero(e.OperationType),
			ResourceType:  ptr.ValueOrZero(e.ResourceType),
			ResourcePath:  ptr.ValueOrZero(e.ResourcePath),
		})
	}
	return events, nil
}

func (k *KeyCloak) lookupUser(ctx context.Context, id string) (*auth.Identity, error) {
	// First, look up by user ID
	resp, err := k.kcClient.GetAdminRealmsRealmUsersUserIdWithResponse(ctx, k.realm, id, nil)
	if err == nil && resp.StatusCode() == http.StatusOK {
		id := k.userToIdentity(*resp.JSON200)
		if id != nil {
			return id, nil
		}
	}

	// next, try lookup by GitHub login
	userLookup, err := k.kcClient.GetAdminRealmsRealmUsersWithResponse(ctx, k.realm, &client.GetAdminRealmsRealmUsersParams{
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
	userLookup, err = k.kcClient.GetAdminRealmsRealmUsersWithResponse(ctx, k.realm, &client.GetAdminRealmsRealmUsersParams{
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
	tokenUrl := kcUrl.JoinPath("realms", cfg.Realm, "/protocol/openid-connect/token")
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
