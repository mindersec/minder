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

// Package dynamic provides the logic for reading and validating JWT tokens
// using a JWKS URL from the token's
package dynamic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"

	stacklok_jwt "github.com/stacklok/minder/internal/auth/jwt"
)

// a subset of the openID well-known configuration for JSON parsing
type openIdConfig struct {
	JwksURI string `json:"jwks_uri"`
}

// Validator dynamically validates JWTs by fetching the key from the well-known OIDC issuer URL.
type Validator struct {
	jwks *jwk.Cache
	aud  string
}

var _ stacklok_jwt.Validator = (*Validator)(nil)

// NewDynamicValidator creates a new instance of the dynamic JWT validator
func NewDynamicValidator(ctx context.Context, aud string) *Validator {
	return &Validator{
		jwks: jwk.NewCache(ctx),
		aud:  aud,
	}
}

// ParseAndValidate implements jwt.Validator.
func (m Validator) ParseAndValidate(tokenString string) (openid.Token, error) {
	// This is based on https://github.com/lestrrat-go/jwx/blob/v2/examples/jwt_parse_with_key_provider_example_test.go

	_, b64payload, _, err := jws.SplitCompact([]byte(tokenString))
	if err != nil {
		return nil, fmt.Errorf("failed to split compact JWT: %w", err)
	}

	jwtPayload := make([]byte, base64.RawStdEncoding.DecodedLen(len(b64payload)))
	if _, err := base64.RawStdEncoding.Decode(jwtPayload, b64payload); err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	parsed, err := jwt.Parse(jwtPayload, jwt.WithVerify(false), jwt.WithToken(openid.New()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT payload: %w", err)
	}
	openIdToken, ok := parsed.(openid.Token)
	if !ok {
		return nil, fmt.Errorf("failed to cast JWT payload to openid.Token")
	}

	// Now that we've got the issuer, we can validate the token
	keySet, err := m.getKeySet(parsed.Issuer())
	if err != nil {
		return nil, fmt.Errorf("failed to get JWK set: %w", err)
	}
	if _, err := jws.Verify([]byte(tokenString), jws.WithKeySet(keySet)); err != nil {
		return nil, fmt.Errorf("failed to verify JWT: %w", err)
	}

	return openIdToken, nil
}

func (m Validator) getKeySet(issuer string) (jwk.Set, error) {
	jwksUrl, err := getJWKSUrlForOpenId(issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS URL from openid: %w", err)
	}
	if err := m.jwks.Register(jwksUrl, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
		return nil, fmt.Errorf("failed to register JWKS URL: %w", err)
	}

	return m.jwks.Get(context.Background(), jwksUrl)
}

func getJWKSUrlForOpenId(issuer string) (string, error) {
	wellKnownUrl := fmt.Sprintf("%s/.well-known/openid-configuration", issuer)

	resp, err := http.Get(wellKnownUrl) // #nosec: G107
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read respons body: %w", err)
	}

	config := openIdConfig{}
	if err := json.Unmarshal(body, &config); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return config.JwksURI, nil
}
