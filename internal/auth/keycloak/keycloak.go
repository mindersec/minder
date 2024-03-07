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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/openfga/go-sdk/oauth2"
	"github.com/openfga/go-sdk/oauth2/clientcredentials"
	"github.com/stacklok/minder/internal/auth"
	serverconfig "github.com/stacklok/minder/internal/config/server"
)

type KeyCloak struct {
	name string
	url  url.URL
	cfg  serverconfig.IdentityConfig
}

func NewKeyCloak(name string, cfg serverconfig.IdentityConfig) *KeyCloak {
	return &KeyCloak{
		name: name,
		url:  cfg.Issuer(),
		cfg:  cfg,
	}
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
	return nil, fmt.Errorf("not implemented")
	/*
		remoteUser, err := k.lookupUser(ctx, id)
		if err != nil {
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
	*/
}

// Validate implements auth.IdentityProvider.
func (k *KeyCloak) Validate(ctx context.Context, token jwt.Token) (*auth.Identity, error) {
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
	client, err := k.newAuthorizedClient(ctx)
	if err != nil {
		return nil, err
	}
	// First, lookup by userID
	idUrl := k.url.JoinPath("realms", "stacklok", "users", id)

	if res, err := client.Get(idUrl.String()); err == nil {
		// TODO: this is a singleton, all the others are lists
		return k.parseUserResponse(res)
	}

	escapedId := url.QueryEscape(id)

	// next, look up by github id
	ghIdUrl := k.url.JoinPath("realms", "stacklok", "users")
	ghIdUrl.RawQuery = fmt.Sprintf("q=gh_id:%s", escapedId)
	if res, err := res, err = client.Get(ghIdUrl.String()); err == nil {
		id, err := k.parseUsersResponse(res)
		if !errors.Is(err, errNotFound) {
			return id, err
		}
		// pass through to the next lookup
	}
	// finally, look up by github login
	ghLoginUrl := k.url.JoinPath("realms", "stacklok", "users")
	ghLoginUrl.RawQuery = fmt.Sprintf("q=gh_login:%s", escapedId)
	res, err := client.Get(ghLoginUrl.String())
	if err != nil {
		return nil, err
	}
	return k.parseUsersResponse(res)
}

func (k *KeyCloak) lookupGithubUser(ctx context.Context, id string) (*auth.Identity, error) {
	return nil, fmt.Errorf("not implemented")
}

func (k *KeyCloak) createKeycloakUser(ctx context.Context, ghUser *auth.Identity) (*auth.Identity, error) {
	return nil, fmt.Errorf("not implemented")
}

func (k *KeyCloak) newAuthorizedClient(ctx context.Context) (*http.Client, error) {
	tokenUrl := k.url.JoinPath("realms/stacklok/protocol/openid-connect/token")
	clientSecret, err := k.cfg.GetClientSecret()
	if err != nil {
		return nil, err
	}
	clientCredentials := clientcredentials.Config{
		ClientID:     k.cfg.ClientId,
		ClientSecret: clientSecret,
		TokenURL:     tokenUrl.String(),
	}

	token, err := clientCredentials.Token(ctx)
	if err != nil {
		return nil, err
	}

	return oauth2.NewClient(ctx, oauth2.StaticTokenSource(token)), nil
}

type keycloakUser struct {
	Id         string `json:"id"`
	Attributes struct {
		GhId    string `json:"gh_id"`
		GhLogin string `json:"gh_login"`
	} `json:"attributes"`
}

func (k *KeyCloak)parseUserResponse(res *http.Response) (*auth.Identity, error) {
	user := keycloakUser{}
	if err := json.NewDecoder(res.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}
	return &auth.Identity{
		UserID:    user.Id,
		HumanName: user.Attributes.GhLogin,
		Provider:  k,
	}, nil
}

func (k *KeyCloak) parseUsersResponse(res *http.Response) (*auth.Identity, error) {
	userList := []keycloakUser{}
	if err := json.NewDecoder(res.Body).Decode(&userList); err != nil {
		return nil, fmt.Errorf("failed to decode user response: %w", err)
	}
	if len(userList) == 0 {
		return nil, errNotFound
	}
	if len(userList) > 1 {
		return nil, fmt.Errorf("multiple users found")
	}

	return &auth.Identity{
		UserID:    userList[0].Id,
		HumanName: userList[0].Attributes.GhLogin,
		Provider:  k,
	}, nil
}
