// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package credentials provides the implementations for the credentials
package credentials

import (
	"golang.org/x/oauth2"
)

// OAuth2TokenCredential is a credential that uses an OAuth2 token
type OAuth2TokenCredential struct {
	token string
}

// NewOAuth2TokenCredential creates a new OAuth2TokenCredential from the token
func NewOAuth2TokenCredential(token string) *OAuth2TokenCredential {
	return &OAuth2TokenCredential{
		token: token,
	}
}

// GetAsOAuth2TokenSource returns the token as an OAuth2 token source
func (o *OAuth2TokenCredential) GetAsOAuth2TokenSource() oauth2.TokenSource {
	return oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: o.token},
	)
}
