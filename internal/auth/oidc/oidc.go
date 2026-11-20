// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package oidc provides a generic OIDC identity provider implementation.
package oidc

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/zitadel/oidc/v3/pkg/client"
	zitadeloidc "github.com/zitadel/oidc/v3/pkg/oidc"

	"github.com/mindersec/minder/internal/auth"
	serverconfig "github.com/mindersec/minder/pkg/config/server"
)

var (
	// ErrMissingSubject indicates a token does not contain a subject claim.
	ErrMissingSubject = errors.New("token has no subject claim")
)

// Provider implements auth.IdentityProvider for generic OIDC compliant providers.
type Provider struct {
	issuerURL url.URL
	name      string
}

// Ensure Provider implements auth.IdentityProvider.
var _ auth.IdentityProvider = (*Provider)(nil)

// NewOIDCProvider creates a new generic OIDC IdentityProvider using OIDC discovery.
func NewOIDCProvider(
	ctx context.Context,
	name string,
	cfg serverconfig.IdentityConfig,
	httpClient *http.Client,
) (*Provider, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	if cfg.IssuerUrl == "" {
		return nil, errors.New("issuer_url is required for OIDC provider")
	}

	discURL, err := url.Parse(cfg.IssuerUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid issuer_url: %w", err)
	}
	wellKnownURL := discURL.JoinPath(zitadeloidc.DiscoveryEndpoint).String()

	expectedIssuer := cmp.Or(cfg.IssuerClaim, cfg.IssuerUrl)

	// Fetch discovery document to validate the provider endpoint and discover issuer
	discovered, err := client.Discover(ctx, expectedIssuer, httpClient, wellKnownURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC configuration at %s: %w", cfg.IssuerUrl, err)
	}

	parsedIssuer, err := url.Parse(discovered.Issuer)
	if err != nil {
		return nil, fmt.Errorf("invalid issuer URL in discovery document: %w", err)
	}

	name = cmp.Or(name, "oidc")

	return &Provider{
		issuerURL: *parsedIssuer,
		name:      name,
	}, nil
}

// String returns the name of this identity provider.
func (p *Provider) String() string {
	return p.name
}

// URL returns the discovered issuer URL.
func (p *Provider) URL() url.URL {
	return p.issuerURL
}

// Validate validates that the supplied token meets basic OIDC expectations and returns
// the extracted Identity. Token signature validation is handled prior by JWT validators.
func (p *Provider) Validate(_ context.Context, token jwt.Token) (*auth.Identity, error) {
	sub, ok := token.Subject()
	if !ok || sub == "" {
		return nil, ErrMissingSubject
	}

	// Try claims in order: preferred_username -> email -> sub
	var humanName string
	if err := token.Get("preferred_username", &humanName); err != nil || humanName == "" {
		if err := token.Get("email", &humanName); err != nil || humanName == "" {
			humanName = sub
		}
	}

	return &auth.Identity{
		UserID:    sub,
		HumanName: humanName,
		Provider:  p,
	}, nil
}

// Resolve implements auth.Resolver. Standard OIDC discovery/tokens provide no admin directory
// lookup API, so Resolve returns auth.ErrNotFound.
func (*Provider) Resolve(_ context.Context, _ string) (*auth.Identity, error) {
	return nil, auth.ErrNotFound
}

// ResolveFederated implements auth.IdentityProvider.
func (*Provider) ResolveFederated(_ context.Context, _, _ string) (*auth.Identity, error) {
	return nil, auth.ErrNotFound
}
