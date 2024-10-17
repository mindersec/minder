// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

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

// GitLabCredential is the interface for credentials used when interacting with GitLab
type GitLabCredential interface {
	RestCredential
	GitCredential
	OAuth2TokenCredential
}
