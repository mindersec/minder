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

// Package v1 for providers provides the public interfaces for the providers
// implemented by minder. The providers are the sources of the data
// that is used by the rules.
package v1

import (
	"net/http"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-containerregistry/pkg/authn"
	"golang.org/x/oauth2"
)

const (
	// CredentialStateSet is the state of a credential when it is set
	CredentialStateSet = "set"
	// CredentialStateUnset is the state of a credential when it is unset
	CredentialStateUnset = "unset"
	// CredentialStateNotApplicable is the state of a credential when it is not applicable
	CredentialStateNotApplicable = "not_applicable"
)

// Credential is the general interface for all credentials
type Credential interface {
}

// RestCredential is the interface for credentials used in REST requests
type RestCredential interface {
	SetAuthorizationHeader(req *http.Request)
}

// GitCredential is the interface for credentials used when performing git operations
type GitCredential interface {
	AddToPushOptions(options *git.PushOptions, owner string)
	AddToCloneOptions(options *git.CloneOptions)
}

// OAuth2TokenCredential is the interface for credentials that are OAuth2 tokens
type OAuth2TokenCredential interface {
	GetAsOAuth2TokenSource() oauth2.TokenSource
}

// GitHubCredential is the interface for credentials used when interacting with GitHub
type GitHubCredential interface {
	RestCredential
	GitCredential
	OAuth2TokenCredential

	GetCacheKey() string
	// as we add new OCI providers this will change to a procedure / mutator, right now it's GitHub specific
	GetAsContainerAuthenticator(owner string) authn.Authenticator
}
