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

// Package merged provides the logic for reading and validating JWT tokens
package merged

import (
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwt/openid"

	stacklok_jwt "github.com/stacklok/minder/internal/auth/jwt"
)

// Validator is a struct that combines multiple JWT validators.
type Validator struct {
	Validators []stacklok_jwt.Validator
}

var _ stacklok_jwt.Validator = (*Validator)(nil)

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
