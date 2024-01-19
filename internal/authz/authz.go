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

// Package authz provides the authorization utilities for minder
package authz

import (
	"fmt"
	"net/http"

	fgaclient "github.com/openfga/go-sdk/client"
	"github.com/openfga/go-sdk/credentials"
	"k8s.io/client-go/transport"

	srvconfig "github.com/stacklok/minder/internal/config/server"
)

// ClientWrapper is a wrapper for the OpenFgaClient.
// It is used to provide a common interface for the client and a way to
// refresh authentication to the authz provider when needed.
type ClientWrapper struct {
	cfg *srvconfig.AuthzConfig
	cli *fgaclient.OpenFgaClient
}

// NewAuthzClient returns a new AuthzClientWrapper
func NewAuthzClient(cfg *srvconfig.AuthzConfig) (*ClientWrapper, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	clicfg := &fgaclient.ClientConfiguration{
		ApiUrl: cfg.ApiUrl,
		Credentials: &credentials.Credentials{
			// We use our own bearer auth round tripper so we can refresh the token
			Method: credentials.CredentialsMethodNone,
		},
	}

	if cfg.StoreID != "" {
		clicfg.StoreId = cfg.StoreID
	}

	if cfg.Auth.Method == "token" {
		rt, err := transport.NewBearerAuthWithRefreshRoundTripper("", cfg.Auth.Token.TokenPath, http.DefaultTransport)
		if err != nil {
			return nil, fmt.Errorf("failed to create bearer auth round tripper: %w", err)
		}

		clicfg.HTTPClient = &http.Client{
			Transport: rt,
		}
	}

	cli, err := fgaclient.NewSdkClient(clicfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create SDK client: %w", err)
	}

	return &ClientWrapper{
		cfg: cfg,
		cli: cli,
	}, nil
}

// GetClient returns the OpenFgaClient
func (a *ClientWrapper) GetClient() *fgaclient.OpenFgaClient {
	// TODO: check if token is expired and refresh it
	// Note that this will probably need a mutex
	return a.cli
}

// GetConfig returns the authz configuration used to build the client
func (a *ClientWrapper) GetConfig() *srvconfig.AuthzConfig {
	return a.cfg
}

// StoreIDProvided returns true if the store ID was provided in the configuration
func (a *ClientWrapper) StoreIDProvided() bool {
	return a.cfg.StoreID != ""
}
