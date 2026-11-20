// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/zitadel/oidc/v3/pkg/client"
	"github.com/zitadel/oidc/v3/pkg/oidc"

	"github.com/mindersec/minder/pkg/config"
)

// OIDCConfig represents the extracted data from the openid-configuration response
type OIDCConfig struct {
	Issuer  string
	JWKSURI url.URL
}

// IdentityConfigWrapper is the configuration for the identity provider
type IdentityConfigWrapper struct {
	Server            IdentityConfig `mapstructure:"server"`
	AdditionalIssuers []string       `mapstructure:"additional_issuers"`
}

// IdentityConfig is the configuration for the identity provider in minder server
type IdentityConfig struct {
	// Type specifies the identity provider type ("keycloak" or "oidc")
	Type string `mapstructure:"type" default:"keycloak"`
	// IssuerUrl is the base URL for calling APIs on the identity server.  Note that this URL
	// is used for direct communication with the identity server, and is not the URL that
	// is included in the JWT tokens.  It is named 'issuer_url' for historical compatibility.
	IssuerUrl string `mapstructure:"issuer_url" default:"http://localhost:8081"`
	// Realm is the realm used by the identity server at IssuerUrl
	Realm string `mapstructure:"realm" default:"stacklok"`
	// IssuerClaim is the claim in the JWT token that identifies the issuer
	IssuerClaim string `mapstructure:"issuer_claim" default:"http://localhost:8081/realms/stacklok"`
	// ClientId is the client ID that identifies the minder server
	ClientId string `mapstructure:"client_id" default:"minder-server"`
	// ClientSecret is the client secret for the minder server.  Prefer using ClientSecretFile
	// instead of this field to avoid storing secrets in config files.
	//nolint:gosec
	ClientSecret string `mapstructure:"client_secret" default:"secret"`
	// ClientSecretFile is the location of a file containing the client secret for the minder server (optional)
	ClientSecretFile string `mapstructure:"client_secret_file"`
	// Audience is the expected audience for JWT tokens (see OpenID spec)
	Audience string `mapstructure:"audience" default:"minder"`
	// Scope is the OAuth scope to request from the identity server to get the specified audience
	Scope string `mapstructure:"scope" default:"minder-audience"`
}

// GetClientSecret returns the minder-server client secret
func (sic *IdentityConfig) GetClientSecret() (string, error) {
	return fileOrArg(sic.ClientSecretFile, sic.ClientSecret, "client secret")
}

// RegisterIdentityFlags registers the flags for the identity server
func RegisterIdentityFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	return config.BindConfigFlag(v, flags, "identity.server.issuer_url", "issuer-url", "",
		"The base URL where the identity server is running", flags.String)
}

// DiscoverOIDCEndpoints fetches the OIDC configuration from the well-known endpoint
func (sic *IdentityConfig) DiscoverOIDCEndpoints(ctx context.Context) (*OIDCConfig, error) {
	discoveryUrl, err := url.Parse(sic.IssuerUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse issuer URL: %w", err)
	}

	// Keycloak specific path for discovery if realm is set and provider type is not oidc
	if sic.Type != "oidc" && sic.Realm != "" {
		discoveryUrl = discoveryUrl.JoinPath("realms", sic.Realm)
	}
	discoveryUrl = discoveryUrl.JoinPath(oidc.DiscoveryEndpoint)

	// Zitadel's OIDC Discovery expects the *issuer* value to match the one provided
	// by the server configuration
	discoveredCfg, err := client.Discover(ctx, sic.IssuerClaim, http.DefaultClient, discoveryUrl.String())
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC configuration: %w", err)
	}

	// The JWKS URIs should be reached via the internal (IssuerURL) hostname, rather
	// than the external URL that might be used by clients. The _iss_ claim from the
	// token should match the issuer Claim.
	jwksURL, err := url.Parse(discoveredCfg.JwksURI)
	if err != nil || jwksURL == nil {
		return nil, fmt.Errorf("failed to parse JWKS URI: %w", err)
	}
	jwksURL.Host = discoveryUrl.Host
	jwksURL.Scheme = discoveryUrl.Scheme

	return &OIDCConfig{
		Issuer:  discoveredCfg.Issuer,
		JWKSURI: *jwksURL,
	}, nil
}

// Issuer returns the URL of the identity server
func (ic *IdentityConfig) Issuer() url.URL {
	u, err := url.Parse(ic.IssuerUrl)
	if err != nil {
		panic("Invalid issuer URL")
	}
	return *u
}
