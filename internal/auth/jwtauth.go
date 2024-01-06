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

package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"

	"github.com/stacklok/minder/internal/util"
)

// RoleInfo contains the role information for a user
type RoleInfo struct {
	RoleID         int32      `json:"role_id"`
	IsAdmin        bool       `json:"is_admin"`
	ProjectID      *uuid.UUID `json:"project_id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
}

// UserPermissions contains the permissions for a user
type UserPermissions struct {
	UserId         int32
	ProjectIds     []uuid.UUID
	Roles          []RoleInfo
	OrganizationId uuid.UUID
	IsStaff        bool
}

// JwtValidator provides the functions to validate a JWT
type JwtValidator interface {
	ParseAndValidate(tokenString string) (openid.Token, error)
}

// JwkSetJwtValidator is a JWT validator that uses a JWK set URL to validate the tokens
type JwkSetJwtValidator struct {
	jwksFetcher KeySetFetcher
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

	token, err := jwt.ParseString(tokenString, jwt.WithKeySet(set), jwt.WithValidate(true), jwt.WithToken(openid.New()))
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
func NewJwtValidator(ctx context.Context, jwksUrl string) (JwtValidator, error) {
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
	}, nil
}

var tokenContextKey struct{}
var userSubjectContextKey struct{}

// GetPermissionsFromContext returns the claims from the context, or an empty default
func GetPermissionsFromContext(ctx context.Context) UserPermissions {
	permissions, ok := ctx.Value(tokenContextKey).(UserPermissions)
	if !ok {
		return UserPermissions{UserId: -1, OrganizationId: uuid.UUID{}}
	}
	return permissions
}

// WithPermissionsContext stores the specified UserClaim in the context.
func WithPermissionsContext(ctx context.Context, claims UserPermissions) context.Context {
	return context.WithValue(ctx, tokenContextKey, claims)
}

// GetUserSubjectFromContext returns the user subject from the context, or nil
func GetUserSubjectFromContext(ctx context.Context) string {
	subject, ok := ctx.Value(userSubjectContextKey).(string)
	if !ok {
		return ""
	}
	return subject
}

// WithUserSubjectContext stores the specified user subject in the context.
func WithUserSubjectContext(ctx context.Context, subject string) context.Context {
	return context.WithValue(ctx, userSubjectContextKey, subject)
}

// GetDefaultProject returns the default project id for the user
func GetDefaultProject(ctx context.Context) (uuid.UUID, error) {
	permissions := GetPermissionsFromContext(ctx)
	if len(permissions.ProjectIds) != 1 {
		return uuid.UUID{}, errors.New("cannot get default project")
	}
	return permissions.ProjectIds[0], nil
}

// UserDetails is a helper struct for getting user details
type UserDetails struct {
	Name  string
	Email string
}

// GetUserDetails is a helper for getting user details such as name and email from the jwt token
func GetUserDetails(ctx context.Context, issuerUrl, clientId string) (*UserDetails, error) {
	t, err := util.GetToken(issuerUrl, clientId)
	if err != nil {
		return nil, err
	}
	parsedURL, err := url.Parse(issuerUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse issuer URL: %w\n", err)
	}
	jwksUrl := parsedURL.JoinPath("realms/stacklok/protocol/openid-connect/certs")
	vldtr, err := NewJwtValidator(ctx, jwksUrl.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch and cache identity provider JWKS: %w\n", err)
	}
	token, err := vldtr.ParseAndValidate(t)
	if err != nil {
		return nil, fmt.Errorf("failed to parse and validate token: %w\n", err)
	}

	return &UserDetails{
		Email: token.Email(),
		Name:  fmt.Sprintf("%s %s", token.GivenName(), token.FamilyName()),
	}, nil
}
