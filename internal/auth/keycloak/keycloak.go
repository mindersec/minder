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
	name      string
	issuerURL url.URL
	realm     string
	cfg       serverconfig.IdentityConfig

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
		name:      name,
		issuerURL: *parsedIssuerUrl,
		realm:     cfg.Realm,
		cfg:       cfg,
		kcClient:  kcClient,
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
	return k.issuerURL
}

// Resolve implements auth.IdentityProvider.
func (k *KeyCloak) Resolve(ctx context.Context, id string) (*auth.Identity, error) {
	// Keycloak doesn't guarantee that user IDs and usernames look different,
	// so try id as a user ID first, then as a username.
	ident, byID := k.lookupByID(ctx, id)
	if ident != nil {
		return ident, nil
	}
	ident, byUsername := k.searchUsers(ctx, "by_username", &client.GetAdminRealmsRealmUsersParams{
		Exact:    ptr.Ptr(true),
		Username: &id,
	})
	if ident != nil {
		return ident, nil
	}

	k.logNoIdentity(ctx, byID, byUsername)
	return nil, auth.ErrNotFound
}

// ResolveFederated implements auth.IdentityProvider.
func (k *KeyCloak) ResolveFederated(ctx context.Context, federatedIdP, id string) (*auth.Identity, error) {
	if federatedIdP != "github" {
		return nil, fmt.Errorf("unsupported federated identity provider %q", federatedIdP)
	}

	// try lookup by GitHub numeric ID (legacy Keycloak behavior)
	ident, byGitHubID := k.searchUsers(ctx, "by_github_id", &client.GetAdminRealmsRealmUsersParams{
		Q: ptr.Ptr(fmt.Sprintf("gh_id:%s", id)),
	})
	if ident != nil {
		return ident, nil
	}

	k.logNoIdentity(ctx, byGitHubID)
	return nil, auth.ErrNotFound
}

// lookupMiss records what one Keycloak user lookup returned when it didn't
// produce an identity.
type lookupMiss struct {
	lookup  string
	err     error
	status  int
	matches int
	// byID marks the lookup by user ID. Keycloak answers it with 404 when the
	// user doesn't exist, while a user search returns 200 and an empty list.
	byID bool
}

// unexpected reports whether the miss points at something other than the
// user not existing, such as the admin API refusing or never receiving the
// request (a gateway that doesn't route /admin answers 404 to searches too).
func (m lookupMiss) unexpected() bool {
	switch {
	case m.err != nil:
		return true
	case m.status == http.StatusOK:
		// A user came back, but without the fields userToIdentity needs.
		return m.byID || m.matches > 0
	case m.status == http.StatusNotFound:
		return !m.byID
	default:
		return true
	}
}

func (m lookupMiss) fields() *zerolog.Event {
	d := zerolog.Dict()
	if m.err != nil {
		return d.AnErr("error", m.err)
	}
	d = d.Int("status", m.status)
	if m.status == http.StatusOK && !m.byID {
		d = d.Int("matches", m.matches)
	}
	return d
}

// lookupByID fetches the user whose Keycloak ID is id.
func (k *KeyCloak) lookupByID(ctx context.Context, id string) (*auth.Identity, lookupMiss) {
	miss := lookupMiss{lookup: "by_id", byID: true}
	resp, err := k.kcClient.GetAdminRealmsRealmUsersUserIdWithResponse(ctx, k.realm, id, nil)
	if err != nil {
		miss.err = err
		return nil, miss
	}
	miss.status = resp.StatusCode()
	if miss.status == http.StatusOK && resp.JSON200 != nil {
		if ident := k.userToIdentity(*resp.JSON200); ident != nil {
			return ident, miss
		}
	}
	return nil, miss
}

// searchUsers returns the identity of the only user matching params.
func (k *KeyCloak) searchUsers(
	ctx context.Context, lookup string, params *client.GetAdminRealmsRealmUsersParams,
) (*auth.Identity, lookupMiss) {
	miss := lookupMiss{lookup: lookup}
	resp, err := k.kcClient.GetAdminRealmsRealmUsersWithResponse(ctx, k.realm, params)
	if err != nil {
		miss.err = err
		return nil, miss
	}
	miss.status = resp.StatusCode()
	if miss.status != http.StatusOK || resp.JSON200 == nil {
		return nil, miss
	}
	miss.matches = len(*resp.JSON200)
	if miss.matches == 1 {
		if ident := k.userToIdentity((*resp.JSON200)[0]); ident != nil {
			return ident, miss
		}
	}
	return nil, miss
}

// logNoIdentity logs every lookup that Resolve or ResolveFederated tried
// before returning auth.ErrNotFound. Without it, a 401, 403 or 404 from the
// admin API looks the same as a user who doesn't exist. It logs at warn level
// if any lookup failed for another reason than the user not existing.
func (k *KeyCloak) logNoIdentity(ctx context.Context, misses ...lookupMiss) {
	level := zerolog.DebugLevel
	for _, m := range misses {
		if m.unexpected() {
			level = zerolog.WarnLevel
		}
	}
	event := zerolog.Ctx(ctx).WithLevel(level).Str("realm", k.realm)
	for _, m := range misses {
		event = event.Dict(m.lookup, m.fields())
	}
	event.Msg("keycloak found no matching user")
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
