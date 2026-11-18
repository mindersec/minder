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

	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/rs/zerolog"
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

var (
	_ auth.IdentityManager  = (*KeyCloak)(nil)
	_ auth.IdentityProvider = (*KeyCloak)(nil)
)

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
	// First, look up by user ID
	resp, err := k.kcClient.GetAdminRealmsRealmUsersUserIdWithResponse(ctx, k.realm, id, nil)
	status, matches := 0, 0
	if err == nil {
		status = resp.StatusCode()
	}
	if status == http.StatusOK {
		matches = 1
		if ident := k.userToIdentity(*resp.JSON200); ident != nil {
			return ident, nil
		}
	}
	k.logNoIdentity(ctx, "user id", err, status, matches)

	// next, try lookup by username
	userLookup, err := k.kcClient.GetAdminRealmsRealmUsersWithResponse(ctx, k.realm, &client.GetAdminRealmsRealmUsersParams{
		Exact:    ptr.Ptr(true),
		Username: &id,
	})
	if ident := k.identityFromList(ctx, "username", userLookup, err); ident != nil {
		return ident, nil
	}

	return nil, auth.ErrNotFound
}

// ResolveFederated implements auth.IdentityProvider.
func (k *KeyCloak) ResolveFederated(ctx context.Context, federatedIdP, id string) (*auth.Identity, error) {
	if federatedIdP != "github" {
		return nil, fmt.Errorf("unsupported federated identity provider %q", federatedIdP)
	}

	// try lookup by GitHub numeric ID (legacy Keycloak behavior)
	userLookup, err := k.kcClient.GetAdminRealmsRealmUsersWithResponse(ctx, k.realm, &client.GetAdminRealmsRealmUsersParams{
		Q: ptr.Ptr(fmt.Sprintf("gh_id:%s", id)),
	})
	if ident := k.identityFromList(ctx, "github id", userLookup, err); ident != nil {
		return ident, nil
	}

	return nil, auth.ErrNotFound
}

// identityFromList returns the identity for a user search that matched
// exactly one user, and logs why it didn't otherwise.
func (k *KeyCloak) identityFromList(
	ctx context.Context, lookup string, resp *client.GetAdminRealmsRealmUsersResponse, err error,
) *auth.Identity {
	status, matches := 0, 0
	if err == nil {
		status = resp.StatusCode()
	}
	if status == http.StatusOK {
		matches = len(*resp.JSON200)
		if matches == 1 {
			if ident := k.userToIdentity((*resp.JSON200)[0]); ident != nil {
				return ident
			}
		}
	}
	k.logNoIdentity(ctx, lookup, err, status, matches)
	return nil
}

// logNoIdentity records why a Keycloak user lookup produced no identity.
// Resolve and ResolveFederated report every failure as auth.ErrNotFound, so
// without this log a 401 or 403 from the admin API (for example, a rotated
// client secret or a service account missing view-users) looks the same as
// a user who doesn't exist.
func (k *KeyCloak) logNoIdentity(ctx context.Context, lookup string, err error, status, matches int) {
	logger := zerolog.Ctx(ctx)
	switch {
	case err != nil:
		logger.Warn().Err(err).Str("realm", k.realm).Str("lookup", lookup).
			Msg("keycloak user lookup request failed")
	case status != http.StatusOK && status != http.StatusNotFound:
		logger.Warn().Str("realm", k.realm).Str("lookup", lookup).Int("status", status).
			Msg("keycloak user lookup returned unexpected status")
	case matches > 1:
		logger.Warn().Str("realm", k.realm).Str("lookup", lookup).Int("matches", matches).
			Msg("keycloak user lookup matched more than one user")
	case matches == 1:
		logger.Warn().Str("realm", k.realm).Str("lookup", lookup).
			Msg("keycloak user lookup matched a user with no id or attributes")
	default:
		logger.Debug().Str("realm", k.realm).Str("lookup", lookup).Int("status", status).
			Msg("keycloak user lookup found no matching user")
	}
}

// Validate implements auth.IdentityProvider.
func (k *KeyCloak) Validate(_ context.Context, token jwt.Token) (*auth.Identity, error) {
	// TODO: implement validating the JWT against the jwks.
	// Note: Currently, JWTs are validated before this method is called within internal/auth/jwt/validator.go.
	var humanStr string
	err := token.Get("preferred_username", &humanStr)
	if err != nil {
		return nil, errors.New("preferred_username not found in token")
	}
	sub, ok := token.Subject()
	if !ok || sub == "" {
		return nil, errors.New("token has no subject")
	}
	return &auth.Identity{
		UserID:    sub,
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
