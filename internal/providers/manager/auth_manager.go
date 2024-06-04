//
// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=./mock/$GOFILE -source=./$GOFILE

package manager

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"

	"github.com/stacklok/minder/internal/db"
	v1 "github.com/stacklok/minder/pkg/providers/v1"
)

// CredentialVerifyParams are the currently supported parameters for credential verification
type CredentialVerifyParams struct {
	RemoteUser string
}

// CredentialVerifyOptFn is a function that sets options for credential verification
type CredentialVerifyOptFn func(*CredentialVerifyParams)

// WithRemoteUser sets the remote user for the credential verification
func WithRemoteUser(remoteUser string) CredentialVerifyOptFn {
	return func(params *CredentialVerifyParams) {
		params.RemoteUser = remoteUser
	}
}

// AuthManager is the interface for managing authentication with provider classes
type AuthManager interface {
	NewOAuthConfig(providerClass db.ProviderClass, cli bool) (*oauth2.Config, error)
	ValidateCredentials(ctx context.Context, providerClass db.ProviderClass, cred v1.Credential, opts ...CredentialVerifyOptFn) error
}

type providerClassAuthManager interface {
}

type providerClassOAuthManager interface {
	ProviderClassManager

	NewOAuthConfig(providerClass db.ProviderClass, cli bool) (*oauth2.Config, error)
	ValidateCredentials(ctx context.Context, cred v1.Credential, params *CredentialVerifyParams) error
}

type authManager struct {
	classTracker
}

// NewAuthManager creates a new AuthManager for managing authentication with providers classes
func NewAuthManager(
	classManagers ...ProviderClassManager,
) (AuthManager, error) {
	classes, err := newClassTracker(classManagers...)
	if err != nil {
		return nil, fmt.Errorf("error creating class tracker: %w", err)
	}

	return &authManager{
		classTracker: *classes,
	}, nil
}

func (a *authManager) NewOAuthConfig(providerClass db.ProviderClass, cli bool) (*oauth2.Config, error) {
	manager, err := a.getClassManager(providerClass)
	if err != nil {
		return nil, fmt.Errorf("error getting class manager: %w", err)
	}

	oauthManager, ok := manager.(providerClassOAuthManager)
	if !ok {
		return nil, fmt.Errorf("class manager does not implement OAuthManager")
	}

	return oauthManager.NewOAuthConfig(providerClass, cli)
}

func (a *authManager) ValidateCredentials(
	ctx context.Context, providerClass db.ProviderClass, cred v1.Credential, opts ...CredentialVerifyOptFn,
) error {
	manager, err := a.getClassManager(providerClass)
	if err != nil {
		return fmt.Errorf("error getting class manager: %w", err)
	}

	oauthManager, ok := manager.(providerClassOAuthManager)
	if !ok {
		return fmt.Errorf("class manager does not implement OAuthManager")
	}

	var verifyParams CredentialVerifyParams

	for _, opt := range opts {
		opt(&verifyParams)
	}

	return oauthManager.ValidateCredentials(ctx, cred, &verifyParams)
}
