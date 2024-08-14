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

// Package jwt provides the logic for reading and validating JWT tokens
package jwt

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

// Validator provides the functions to validate a JWT
type Validator interface {
	ParseAndValidate(tokenString string) (openid.Token, error)
}

// JwkSetJwtValidator is a JWT validator that uses a JWK set URL to validate the tokens
type JwkSetJwtValidator struct {
	jwksFetcher KeySetFetcher
	iss         string
	aud         string
}

// KeySetFetcher provides the functions to fetch a JWK set
type KeySetFetcher interface {
	GetKeySet() (jwk.Set, error)
}

// KeySetCache is a KeySetFetcher that fetches the JWK set from a cache
type KeySetCache struct {
	ctx       context.Context
	jwksUrl   string
	jwksCache *jwk.Cache
}

// GetKeySet returns the caches JWK set
func (k *KeySetCache) GetKeySet() (jwk.Set, error) {
	return k.jwksCache.Get(k.ctx, k.jwksUrl)
}

// ParseAndValidate validates a token string and returns an openID token, or an error if the token is invalid
func (j *JwkSetJwtValidator) ParseAndValidate(tokenString string) (openid.Token, error) {
	set, err := j.jwksFetcher.GetKeySet()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseString(
		tokenString,
		jwt.WithKeySet(set),
		jwt.WithIssuer(j.iss),
		jwt.WithValidate(true),
		jwt.WithToken(openid.New()),
		jwt.WithAudience(j.aud))
	if err != nil {
		return nil, err
	}

	openIdToken, ok := token.(openid.Token)
	if !ok {
		return nil, fmt.Errorf("provided token was not an OpenID token")
	}

	if openIdToken.Subject() == "" {
		return nil, fmt.Errorf("provided token is missing required subject claim")
	}

	return openIdToken, nil
}

// NewJwtValidator creates a new JWT validator that uses a JWK set URL to validate the tokens
func NewJwtValidator(ctx context.Context, jwksUrl string, issUrl string, aud string) (Validator, error) {
	// Cache the JWK set
	// The cache will refresh every 15 minutes by default
	jwks := jwk.NewCache(ctx)
	err := jwks.Register(jwksUrl)
	if err != nil {
		return nil, err
	}

	// Refresh the JWKS once before starting
	_, err = jwks.Refresh(ctx, jwksUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh identity provider JWKS: %s\n", err)
	}

	keySetCache := KeySetCache{
		ctx:       ctx,
		jwksUrl:   jwksUrl,
		jwksCache: jwks,
	}
	return &JwkSetJwtValidator{
		jwksFetcher: &keySetCache,
		iss:         issUrl,
		aud:         aud,
	}, nil
}

type userTokenContextKeyType struct{}

var userTokenContextKey userTokenContextKeyType

// GetUserSubjectFromContext returns the user subject from the context, or nil
func GetUserSubjectFromContext(ctx context.Context) string {
	token, ok := ctx.Value(userTokenContextKey).(openid.Token)
	if !ok {
		return ""
	}
	return token.Subject()
}

// GetUserClaimFromContext returns the specified claim from the user subject in
// the context if found and of the correct type
func GetUserClaimFromContext[T any](ctx context.Context, claim string) (T, bool) {
	var ret T
	token, ok := ctx.Value(userTokenContextKey).(openid.Token)
	if !ok {
		return ret, false
	}
	data, ok := token.Get(claim)
	if !ok {
		return ret, false
	}
	ret, ok = data.(T)
	return ret, ok
}

// WithAuthTokenContext stores the specified user-identifying token in the context.
func WithAuthTokenContext(ctx context.Context, token openid.Token) context.Context {
	return context.WithValue(ctx, userTokenContextKey, token)
}

// GetUserEmailFromContext returns the user email from the context, or an empty string
func GetUserEmailFromContext(ctx context.Context) (string, error) {
	token, ok := ctx.Value(userTokenContextKey).(openid.Token)
	if !ok {
		return "", fmt.Errorf("no user token in context")
	}
	return token.Email(), nil
}

// GetUserTokenFromContext returns the user token from the context
func GetUserTokenFromContext(ctx context.Context) (openid.Token, error) {
	token, ok := ctx.Value(userTokenContextKey).(openid.Token)
	if !ok {
		return nil, fmt.Errorf("no user token in context")
	}
	return token, nil
}
