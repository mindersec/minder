// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
