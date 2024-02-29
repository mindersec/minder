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
	"net/url"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stacklok/minder/internal/auth"
)

type KeyCloak struct {
	name string
	url  url.URL
}

func NewKeyCloak(name string, url url.URL) *KeyCloak {
	return &KeyCloak{
		name: name,
		url:  url,
	}
}

var _ auth.IdentityProvider = (*KeyCloak)(nil)

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
	panic("unimplemented")
	// TODO: resolve the entity using the KeyCloak API.
}

// Validate implements auth.IdentityProvider.
func (k *KeyCloak) Validate(ctx context.Context, token jwt.Token) (*auth.Identity, error) {
	// TODO: implement validating the JWT against the jwks.

	humanName, ok := token.Get("preferred_username")
	if !ok {
		return nil, errors.New("preferred_username not found in token")
	}
	return &auth.Identity{
		UserID:    token.Subject(),
		HumanName: humanName(),
		Provider:  k,
	}, nil
}

var _ auth.IdentityProvider = (*KeyCloak)(nil)
