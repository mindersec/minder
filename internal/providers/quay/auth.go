// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package quay

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"

	"github.com/mindersec/minder/internal/db"
	"github.com/mindersec/minder/internal/providers/manager"
	provv1 "github.com/mindersec/minder/pkg/providers/v1"
)

// NewOAuthConfig implements the providerClassOAuthManager interface.
// Quay only supports the user-input (token) authorization flow, so there
// is no OAuth2 authorization code flow to configure.
func (*providerClassManager) NewOAuthConfig(_ db.ProviderClass, _ bool) (*oauth2.Config, error) {
	return nil, fmt.Errorf("quay provider does not support the OAuth2 authorization code flow")
}

// ValidateCredentials implements the providerClassOAuthManager interface.
// Quay credentials are supplied directly by the user (a Quay.io Robot
// account token) via the user-input flow, so this always receives a raw
// token string, never an OAuth2-derived credential.
func (*providerClassManager) ValidateCredentials(
	_ context.Context, cred provv1.Credential, _ *manager.CredentialVerifyParams,
) error {
	switch c := cred.(type) {
	case string:
		if c == "" {
			return fmt.Errorf("token must not be empty")
		}
	default:
		return fmt.Errorf("invalid credential type: %T", cred)
	}

	return nil
}
