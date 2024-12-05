// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package noop provides a no-op implementation of the Validator interface
package noop

import (
	"github.com/lestrrat-go/jwx/v2/jwt/openid"

	"github.com/mindersec/minder/internal/auth/jwt"
)

type noopJwtValidator struct {
	// Subject is the subject of the token that will be returned by ParseAndValidate
	Subject string
}

// NewJwtValidator returns a new instance of the no-op JWT validator
func NewJwtValidator(subject string) jwt.Validator {
	return &noopJwtValidator{
		Subject: subject,
	}
}

// ParseAndValidate returns a token with the subject set to the subject of the no-op JWT validator
func (n *noopJwtValidator) ParseAndValidate(_ string) (openid.Token, error) {
	tok := openid.New()
	if err := tok.Set("sub", n.Subject); err != nil {
		return nil, err
	}

	return tok, nil
}
