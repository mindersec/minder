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
	"sync"
	"time"

	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
	"github.com/lestrrat-go/jwx/v3/jws/jwsbb"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/lestrrat-go/jwx/v3/jwt/openid"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/singleflight"

	minder_jwt "github.com/mindersec/minder/internal/auth/jwt"
)

// a subset of the openID well-known configuration for JSON parsing
type openIdConfig struct {
	JwksURI string `json:"jwks_uri"`
}

var (
	cachedIssuers metric.Int64Counter
	deniedIssuers metric.Int64Counter
	dynamicAuths  metric.Int64Counter
	metricsInit   sync.Once
)

// Validator dynamically validates JWTs by fetching the key from the well-known OIDC issuer URL.
type Validator struct {
	jwks           *jwk.Cache
	aud            string
	allowedIssuers []string

	// certsMu guards allowedCerts. It is only held for map access, never
	// across network calls or jwks.Register (which calls IsAllowed).
	certsMu      sync.RWMutex
	allowedCerts map[string]struct{}
	// mu guards jwksURLs. It is only held for map access, never across
	// network calls or jwks.Register.
	mu sync.RWMutex
	// jwksURLs maps an allowed issuer to the JWKS URL from its discovery doc.
	jwksURLs map[string]string
	// discovery makes concurrent first requests for an issuer share one
	// discovery and registration.
	discovery singleflight.Group
}

var _ minder_jwt.Validator = (*Validator)(nil)

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

	ret := &Validator{
		aud:            aud,
		allowedIssuers: issuers,
		allowedCerts:   make(map[string]struct{}),
		jwksURLs:       make(map[string]string),
	}
	var err error
	ret.jwks, err = jwk.NewCache(ctx, httprc.NewClient(httprc.WithWhitelist(ret)))
	if err != nil {
		zerolog.Ctx(context.Background()).Warn().Err(err).Msg("Failed to create JWK cache")
		return nil
	}
	return ret
}

// ParseAndValidate implements jwt.Validator.
func (m *Validator) ParseAndValidate(tokenString string) (openid.Token, error) {
	if dynamicAuths != nil {
		dynamicAuths.Add(context.Background(), 1)
	}
	// This is based on https://github.com/lestrrat-go/jwx/blob/develop/v3/examples/jwt_parse_with_key_provider_example_test.go

	_, b64payload, _, err := jwsbb.SplitCompact([]byte(tokenString))
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

	iss, ok := parsed.Issuer()
	if !ok || iss == "" {
		return nil, fmt.Errorf("provided token is missing required issuer claim")
	}
	// Now that we've got the issuer, we can validate the token
	keySet, err := m.getKeySet(iss)
	if err != nil {
		return nil, fmt.Errorf("failed to get JWK set: %w", err)
	}
	if _, err := jws.Verify([]byte(tokenString), jws.WithKeySet(keySet)); err != nil {
		return nil, fmt.Errorf("failed to verify JWT: %w", err)
	}

	return openIdToken, nil
}

// getKeySet fetches the JWK set for the given issuer and adds it to the cache.
// Once discovery succeeds for an issuer, its JWKS URL is cached and the
// well-known URL is not fetched again. Failed discovery is retried on the next
// request. This implementation is not ideal:
// - we never re-fetch the well-known URL, so if the JWKS URL changes, we won't pick it up
// - we also never invalidate JWKS URLs in the cache or allow-list.
// Adding the above logic would add a lot of complexity for only theoretical benefit.
//
// IMPORTANT CONSTRAINT: As the issuer is a potentially attacker-controlled arbitrary value
// this function should not store any per-issuer state except for a bounded set of URLs
// (such as the allowlist). The issuer to JWKS URL cache is only written after the issuer
// is checked against allowedIssuers, so it is bounded by that list.
func (m *Validator) getKeySet(issuer string) (jwk.Set, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if !slices.Contains(m.allowedIssuers, issuer) {
		if deniedIssuers != nil {
			deniedIssuers.Add(ctx, 1)
		}
		return nil, fmt.Errorf("issuer %s is not allowed", issuer)
	}
	m.mu.RLock()
	jwksUrl, ok := m.jwksURLs[issuer]
	m.mu.RUnlock()
	if !ok {
		v, err, _ := m.discovery.Do(issuer, func() (any, error) {
			return m.registerIssuer(ctx, issuer)
		})
		if err != nil {
			return nil, err
		}
		jwksUrl = v.(string)
	}
	return m.jwks.Lookup(ctx, jwksUrl)
}

// registerIssuer fetches the issuer's JWKS URL, registers it with the JWK
// cache and caches it for the issuer. Nothing is cached on failure, so the
// next request retries.
func (m *Validator) registerIssuer(ctx context.Context, issuer string) (string, error) {
	jwksUrl, err := getJWKSUrlForOpenId(ctx, issuer)
	if err != nil {
		return "", fmt.Errorf("failed to fetch JWKS URL from openid: %w", err)
	}
	// A failed Register can still leave the URL registered (e.g. if the first
	// fetch times out), so check before registering again.
	if !m.jwks.IsRegistered(ctx, jwksUrl) {
		if cachedIssuers != nil {
			cachedIssuers.Add(ctx, 1)
		}
		m.certsMu.Lock()
		m.allowedCerts[jwksUrl] = struct{}{}
		m.certsMu.Unlock()
		if err := m.jwks.Register(ctx, jwksUrl, jwk.WithMinInterval(15*time.Minute)); err != nil {
			return "", fmt.Errorf("failed to register JWKS URL: %w", err)
		}
	}
	m.mu.Lock()
	m.jwksURLs[issuer] = jwksUrl
	m.mu.Unlock()
	return jwksUrl, nil
}

func getJWKSUrlForOpenId(ctx context.Context, issuer string) (string, error) {
	wellKnownUrl := fmt.Sprintf("%s/.well-known/openid-configuration", issuer)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnownUrl, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req) // #nosec: G107
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	config := openIdConfig{}
	if err := json.Unmarshal(body, &config); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return config.JwksURI, nil
}

// IsAllowed implements httprc.Whitelist
func (m *Validator) IsAllowed(url string) bool {
	m.certsMu.RLock()
	defer m.certsMu.RUnlock()
	_, ok := m.allowedCerts[url]
	return ok
}
