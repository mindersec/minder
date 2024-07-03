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

// Package noop provides a no-op implementation of the Validator interface
package noop

import (
	"github.com/lestrrat-go/jwx/v2/jwt/openid"

	"github.com/stacklok/minder/internal/auth/jwt"
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
