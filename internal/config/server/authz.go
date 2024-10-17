// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"os"

	"github.com/go-playground/validator/v10"
)

// AuthzConfig is the configuration for minder's authorization
type AuthzConfig struct {
	// ApiUrl is the URL to the authorization server
	ApiUrl string `mapstructure:"api_url" validate:"required"`
	// StoreName is the name of the store to use for authorization
	StoreName string `mapstructure:"store_name" default:"minder" validate:"required_without=StoreID"`
	// StoreID is the ID of the store to use for authorization
	StoreID string `mapstructure:"store_id" default:"" validate:"required_without=StoreName"`
	// ModelID is the ID of the model to use for authorization
	ModelID string `mapstructure:"model_id" default:""`
	// Auth is the authentication configuration for the authorization server
	Auth OpenFGAAuth `mapstructure:"auth" validate:"required"`
}

// Validate validates the Authz configuration
func (a *AuthzConfig) Validate() error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(a); err != nil {
		return err
	}

	return a.Auth.Validate()
}

// OpenFGAAuth contains the authentication configuration for OpenFGA
type OpenFGAAuth struct {
	// Method is the authentication method to use
	Method string `mapstructure:"method" default:"none" validate:"oneof=token none"`

	// Token is the configuration for OpenID Connect authentication
	Token TokenAuth `mapstructure:"token"`
}

// Validate validates the OpenFGAAuth configuration
func (o *OpenFGAAuth) Validate() error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(o); err != nil {
		return err
	}

	if o.Method == "none" {
		return nil
	}

	return o.Token.Validate()
}

// TokenAuth contains the configuration for token authentication
type TokenAuth struct {
	// TokenPath is the path to the token to use for authentication.
	// defaults to the kubernetes service account token
	//nolint:lll
	TokenPath string `mapstructure:"token_path" default:"/var/run/secrets/kubernetes.io/serviceaccount/token"`
}

// Validate validates the TokenAuth configuration
func (t *TokenAuth) Validate() error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	return validate.Struct(t)
}

// ReadToken reads the token from the configured path
func (t *TokenAuth) ReadToken() (string, error) {
	tok, err := os.ReadFile(t.TokenPath)
	if err != nil {
		return "", err
	}

	return string(tok), nil
}
