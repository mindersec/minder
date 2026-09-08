// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package dockerhub

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers/manager"
	provv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// NewOAuthConfig implements the providerClassOAuthManager interface.
// DockerHub only supports the user-input (token) authorization flow, so
// there is no OAuth2 authorization code flow to configure.
func (*providerClassManager) NewOAuthConfig(_ db.ProviderClass, _ bool) (*oauth2.Config, error) {
	return nil, fmt.Errorf("dockerhub provider does not support the OAuth2 authorization code flow")
}

// ValidateCredentials implements the providerClassOAuthManager interface.
// DockerHub credentials are supplied directly by the user (a Docker Hub
// Personal Access Token or account password) rather than obtained via an
// OAuth2 exchange, so validation is limited to a basic sanity check.
func (*providerClassManager) ValidateCredentials(
	_ context.Context, cred provv1.Credential, _ *manager.CredentialVerifyParams,
) error {
	switch c := cred.(type) {
	case provv1.OAuth2TokenCredential:
		if _, err := c.GetAsOAuth2TokenSource().Token(); err != nil {
			return fmt.Errorf("cannot get token from credential: %w", err)
		}
	case string:
		if c == "" {
			return fmt.Errorf("token must not be empty")
		}
	default:
		return fmt.Errorf("invalid credential type: %T", cred)
	}

	return nil
}
