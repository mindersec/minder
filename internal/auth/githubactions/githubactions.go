// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package githubactions provides an implementation of the GitHub IdentityProvider.
package githubactions

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwt"

	"github.com/mindersec/minder/internal/auth"
)

// GitHubActions is an implementation of the auth.IdentityProvider interface.
type GitHubActions struct {
}

var _ auth.IdentityProvider = (*GitHubActions)(nil)
var _ auth.Resolver = (*GitHubActions)(nil)

var ghIssuerUrl = url.URL{
	Scheme: "https",
	Host:   "token.actions.githubusercontent.com",
}

// String implements auth.IdentityProvider.
func (*GitHubActions) String() string {
	return "githubactions"
}

// URL implements auth.IdentityProvider.
func (*GitHubActions) URL() url.URL {
	return ghIssuerUrl
}

// Resolve implements auth.IdentityProvider.
func (gha *GitHubActions) Resolve(_ context.Context, id string) (*auth.Identity, error) {
	// GitHub Actions subjects look like:
	// repo:evankanderson/actions-id-token-testing:ref:refs/heads/main
	// however, OpenFGA does not allow the "#" or ":" characters in the subject:
	// https://github.com/openfga/openfga/blob/main/pkg/tuple/tuple.go#L34
	return &auth.Identity{
		UserID:    strings.ReplaceAll(id, ":", "+"),
		HumanName: strings.ReplaceAll(id, "+", ":"),
		Provider:  gha,
	}, nil
}

// Validate implements auth.IdentityProvider.
func (gha *GitHubActions) Validate(_ context.Context, token jwt.Token) (*auth.Identity, error) {
	expectedUrl := gha.URL()
	if token.Issuer() != expectedUrl.String() {
		return nil, errors.New("token issuer is not the expected issuer")
	}
	return &auth.Identity{
		UserID:    strings.ReplaceAll(token.Subject(), ":", "+"),
		HumanName: token.Subject(),
		Provider:  gha,
	}, nil
}
