// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package dockerhub

import (
	"context"
	"fmt"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers/manager"
	provv1 "github.com/mindersec/minder/pkg/providers/v1"
	"golang.org/x/oauth2"
)

// NewOAuthConfig is not supported by DockerHub.
func (*providerClassManager) NewOAuthConfig(
	_ db.ProviderClass,
	_ bool,
) (*oauth2.Config, error) {
	return nil, fmt.Errorf("dockerhub provider does not support the OAuth2 authorization code flow")
}

// ValidateCredentials checks that the provided DockerHub credential is valid.
func (*providerClassManager) ValidateCredentials(
	_ context.Context,
	cred provv1.Credential,
	_ *manager.CredentialVerifyParams,
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