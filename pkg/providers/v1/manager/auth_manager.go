//
// Copyright 2024 Stacklok, Inc.
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

package manager

import (
	"context"

	"golang.org/x/oauth2"

	"github.com/mindersec/minder/pkg/providers/v1"
)

//go:generate go run go.uber.org/mock/mockgen -package mock_$GOPACKAGE -destination=../../../../internal/providers/manager/mock/if_$GOFILE -source=./$GOFILE

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
	NewOAuthConfig(providerClass string, cli bool) (*oauth2.Config, error)
	ValidateCredentials(ctx context.Context, providerClass string, cred v1.Credential, opts ...CredentialVerifyOptFn) error
}
