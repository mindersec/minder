// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package merged provides the logic for reading and validating JWT tokens
package merged

import (
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwt/openid"

	minder_jwt "github.com/mindersec/minder/internal/auth/jwt"
)

// Validator is a struct that combines multiple JWT validators.
type Validator struct {
	Validators []minder_jwt.Validator
}

var _ minder_jwt.Validator = (*Validator)(nil)

// ParseAndValidate implements jwt.Validator.
func (m Validator) ParseAndValidate(tokenString string) (openid.Token, error) {
	for _, v := range m.Validators {
		t, err := v.ParseAndValidate(tokenString)
		if err == nil {
			return t, nil
		}
	}
	return nil, fmt.Errorf("no validator could parse and validate the token")
}
