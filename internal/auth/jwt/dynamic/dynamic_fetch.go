// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package dynamic provides the logic for reading and validating JWT tokens
// using a JWKS URL from the token's `iss` claim.
package dynamic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/lestrrat-go/jwx/v2/jwt/openid"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	stacklok_jwt "github.com/mindersec/minder/internal/auth/jwt"
)

// a subset of the openID well-known configuration for JSON parsing
type openIdConfig struct {
	JwksURI string `json:"jwks_uri"`
}

var cachedIssuers metric.Int64Counter
var deniedIssuers metric.Int64Counter
var dynamicAuths metric.Int64Counter
var metricsInit sync.Once

// Validator dynamically validates JWTs by fetching the key from the well-known OIDC issuer URL.
type Validator struct {
	jwks           *jwk.Cache
	aud            string
	allowedIssuers []string
}

var _ stacklok_jwt.Validator = (*Validator)(nil)

// NewDynamicValidator creates a new instance of the dynamic JWT validator
func NewDynamicValidator(ctx context.Context, aud string, issuers []string) *Validator {
	metricsInit.Do(func() {
		meter := otel.Meter("minder")
		var err error
		cachedIssuers, err = meter.Int64Counter("dynamic_jwt.cached_issuers",
			metric.WithDescription("Number of cached issuers for dynamic JWT validation"),
			metric.WithUnit("count"),
		)
		if err != nil {
			zerolog.Ctx(context.Background()).Warn().Err(err).Msg("Creating gauge for cached issuers failed")
		}
		deniedIssuers, err = meter.Int64Counter("dynamic_jwt.denied_issuers",
			metric.WithDescription("Number of denied issuers for dynamic JWT validation"),
			metric.WithUnit("count"),
		)
		if err != nil {
			zerolog.Ctx(context.Background()).Warn().Err(err).Msg("Creating gauge for denied issuers failed")
		}
		dynamicAuths, err = meter.Int64Counter("dynamic_jwt.auths",
			metric.WithDescription("Number of dynamic JWT authentications"),
			metric.WithUnit("count"),
		)
		if err != nil {
			zerolog.Ctx(context.Background()).Warn().Err(err).Msg("Creating gauge for dynamic JWT authentications failed")
		}
	})
	return &Validator{
		jwks:           jwk.NewCache(ctx),
		aud:            aud,
		allowedIssuers: issuers,
	}
}

// ParseAndValidate implements jwt.Validator.
func (m Validator) ParseAndValidate(tokenString string) (openid.Token, error) {
	if dynamicAuths != nil {
		dynamicAuths.Add(context.Background(), 1)
	}
	// This is based on https://github.com/lestrrat-go/jwx/blob/v2/examples/jwt_parse_with_key_provider_example_test.go

	_, b64payload, _, err := jws.SplitCompact([]byte(tokenString))
	if err != nil {
		return nil, fmt.Errorf("failed to split compact JWT: %w", err)
	}

	jwtPayload := make([]byte, base64.RawStdEncoding.DecodedLen(len(b64payload)))
	if _, err := base64.RawStdEncoding.Decode(jwtPayload, b64payload); err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	parsed, err := jwt.Parse(jwtPayload,
		jwt.WithVerify(false), jwt.WithToken(openid.New()), jwt.WithAudience(m.aud))
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
	if !slices.Contains(m.allowedIssuers, issuer) {
		if deniedIssuers != nil {
			deniedIssuers.Add(context.Background(), 1)
		}
		return nil, fmt.Errorf("issuer %s is not allowed", issuer)
	}
	jwksUrl, err := getJWKSUrlForOpenId(issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS URL from openid: %w", err)
	}
	ret, err := m.jwks.Get(context.Background(), jwksUrl)
	if err == nil {
		return ret, err
	}
	// There's no nice way to check this error, which contains dynamic content.  :-(
	if strings.Contains(err.Error(), "is not registered") {
		if cachedIssuers != nil {
			cachedIssuers.Add(context.Background(), 1)
		}
		if err := m.jwks.Register(jwksUrl, jwk.WithMinRefreshInterval(15*time.Minute)); err != nil {
			return nil, fmt.Errorf("failed to register JWKS URL: %w", err)
		}

		return m.jwks.Get(context.Background(), jwksUrl)
	}
	return nil, err
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
		return "", fmt.Errorf("Failed to read response body: %w", err)
	}

	config := openIdConfig{}
	if err := json.Unmarshal(body, &config); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return config.JwksURI, nil
}
