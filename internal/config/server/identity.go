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

package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	"github.com/stacklok/minder/internal/config"
)

// IdentityConfigWrapper is the configuration for the identity provider
type IdentityConfigWrapper struct {
	Server IdentityConfig `mapstructure:"server"`
}

// IdentityConfig is the configuration for the identity provider in minder server
type IdentityConfig struct {
	// IssuerUrl is the base URL where the identity server is running
	IssuerUrl string `mapstructure:"issuer_url" default:"http://localhost:8081"`
	// ClientId is the client ID that identifies the minder server
	ClientId string `mapstructure:"client_id" default:"minder-server"`
	// ClientSecret is the client secret for the minder server
	ClientSecret string `mapstructure:"client_secret" default:"secret"`
	// ClientSecretFile is the location of a file containing the client secret for the minder server (optional)
	ClientSecretFile string `mapstructure:"client_secret_file"`
}

// GetClientSecret returns the minder-server client secret
func (sic *IdentityConfig) GetClientSecret() (string, error) {
	if sic.ClientSecretFile != "" {
		data, err := os.ReadFile(filepath.Clean(sic.ClientSecretFile))
		if err != nil {
			return "", fmt.Errorf("failed to read minder secret from file: %w", err)
		}
		return string(data), nil
	}
	return sic.ClientSecret, nil
}

// RegisterIdentityFlags registers the flags for the identity server
func RegisterIdentityFlags(v *viper.Viper, flags *pflag.FlagSet) error {
	return config.BindConfigFlag(v, flags, "identity.server.issuer_url", "issuer-url", "",
		"The base URL where the identity server is running", flags.String)
}

// Path returns a URL for the given path on the identity server
func (sic *IdentityConfig) Path(path string) (*url.URL, error) {
	parsedUrl, err := url.Parse(sic.IssuerUrl)
	if err != nil {
		return nil, err
	}
	return parsedUrl.JoinPath(path), nil
}

func (sic *IdentityConfig) getClient(ctx context.Context) (*http.Client, error) {
	tokenUrl, err := sic.Path("realms/stacklok/protocol/openid-connect/token")
	if err != nil {
		return nil, err
	}

	clientSecret, err := sic.GetClientSecret()
	if err != nil {
		return nil, err
	}

	clientCredentials := clientcredentials.Config{
		ClientID:     sic.ClientId,
		ClientSecret: clientSecret,
		TokenURL:     tokenUrl.String(),
	}

	token, err := clientCredentials.Token(ctx)
	if err != nil {
		return nil, err
	}

	return oauth2.NewClient(ctx, oauth2.StaticTokenSource(token)), nil
}

// Do sends an HTTP request to the identity server, using the configured client credentials.
func (sic *IdentityConfig) Do(
	ctx context.Context, method string, path string, query url.Values, body io.Reader) (*http.Response, error) {
	parsedUrl, err := sic.Path(path)
	if err != nil {
		return nil, err
	}
	parsedUrl.RawQuery = query.Encode()

	req, err := http.NewRequest(method, parsedUrl.String(), body)
	if err != nil {
		return nil, err
	}

	client, err := sic.getClient(ctx)
	if err != nil {
		return nil, err
	}

	return client.Do(req)
}

// Issuer returns the URL of the identity server
func (ic *IdentityConfig) Issuer() url.URL {
	u, err := url.Parse(ic.IssuerUrl)
	if err != nil {
		panic("Invalid issuer URL")
	}
	return *u
}
