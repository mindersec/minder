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

	"github.com/mindersec/minder/internal/db"
	v1 "github.com/mindersec/minder/pkg/providers/v1"
	mgrif "github.com/mindersec/minder/pkg/providers/v1/manager"
)

type providerClassOAuthManager interface {
	mgrif.ProviderClassManager

	NewOAuthConfig(providerClass string, cli bool) (*oauth2.Config, error)
	ValidateCredentials(ctx context.Context, cred v1.Credential, params *mgrif.CredentialVerifyParams) error
}

type authManager struct {
	classTracker
}

// NewAuthManager creates a new AuthManager for managing authentication with providers classes
func NewAuthManager(
	classManagers ...mgrif.ProviderClassManager,
) (mgrif.AuthManager, error) {
	classes, err := newClassTracker(classManagers...)
	if err != nil {
		return nil, fmt.Errorf("error creating class tracker: %w", err)
	}

	return &authManager{
		classTracker: *classes,
	}, nil
}

func (a *authManager) NewOAuthConfig(providerClass string, cli bool) (*oauth2.Config, error) {
	manager, err := a.getClassManager(db.ProviderClass(providerClass))
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
	ctx context.Context, providerClass string, cred v1.Credential, opts ...mgrif.CredentialVerifyOptFn,
) error {
	manager, err := a.getClassManager(db.ProviderClass(providerClass))
	if err != nil {
		return fmt.Errorf("error getting class manager: %w", err)
	}

	oauthManager, ok := manager.(providerClassOAuthManager)
	if !ok {
		return fmt.Errorf("class manager does not implement OAuthManager")
	}

	var verifyParams mgrif.CredentialVerifyParams

	for _, opt := range opts {
		opt(&verifyParams)
	}

	return oauthManager.ValidateCredentials(ctx, cred, &verifyParams)
}
