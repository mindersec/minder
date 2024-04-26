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

package auth

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwt"
)

type identityTestCase struct {
	url   string
	id    string
	sub   string
	human string
}

func TestIdentityClient_Register(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		providers map[string]string
		tests     []identityTestCase
	}{{
		name:      "Single provider",
		providers: map[string]string{"": "github"},
		tests: []identityTestCase{
			{url: "https:///token", id: "github", sub: "github", human: "GITHUB"},
		},
	}, {
		name:      "Two providers",
		providers: map[string]string{"": "github", "accounts.google.com": "google"},
		tests: []identityTestCase{
			{url: "https:///token", id: "github", sub: "github", human: "GITHUB"},
			{url: "https://accounts.google.com/token", id: "accounts.google.com/google", sub: "google", human: "accounts.google.com/GOOGLE"},
		},
	}, {
		name:      "No default provider",
		providers: map[string]string{"github.com": "github", "accounts.google.com": "google"},
		tests: []identityTestCase{
			{url: "https://github.com/token", id: "github.com/github", sub: "github", human: "github.com/GITHUB"},
			{url: "https://accounts.google.com/token", id: "accounts.google.com/google", sub: "google", human: "accounts.google.com/GOOGLE"},
		},
	}}

	ValidateClient := func(t *testing.T, c *IdentityClient, tc []identityTestCase) {
		t.Helper()
		ctx := context.Background()
		for _, tc := range tc {
			id, err := c.Resolve(ctx, tc.id)
			if err != nil {
				t.Fatalf("Resolve(%q) = %v", tc.id, err)
			}
			if id.Human() != tc.human {
				t.Errorf("Resolve(%q) = %v; want %q", tc.id, id.HumanName, tc.human)
			}

			userJwt := jwt.New()
			if err := userJwt.Set("sub", tc.sub); err != nil {
				t.Fatalf("jwt.Set(sub) = %v", err)
			}
			if err := userJwt.Set("iss", tc.url); err != nil {
				t.Fatalf("jwt.Set(iss) = %v", err)
			}

			id, err = c.Validate(ctx, userJwt)
			if err != nil {
				t.Fatalf("Validate(%q) = %v", tc.sub, err)
			}
			if id.Human() != tc.human {
				t.Errorf("Validate(%q) = %v; want %q", tc.sub, id.HumanName, tc.human)
			}
			if id.String() != tc.id {
				t.Errorf("Validate(%q) = %v; want %q", tc.sub, id.String(), tc.id)
			}
		}
	}

	for _, tt := range tests {
		providers := make([]IdentityProvider, 0, len(tt.providers))
		for name, wantId := range tt.providers {
			providers = append(providers, &StaticIDP{
				name:   name,
				wantId: wantId,
				human:  strings.ToUpper(wantId),
			})
		}
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := NewIdentityClient(providers...)
			if err != nil {
				t.Fatalf("NewIdentityClient() = %v", err)
			}

			ValidateClient(t, c, tt.tests)
		})
		t.Run(tt.name+"-register", func(t *testing.T) {
			t.Parallel()
			c, err := NewIdentityClient()
			if err != nil {
				t.Fatalf("NewIdentityClient() = %v", err)
			}
			for _, p := range providers {
				if err := c.Register(p); err != nil {
					t.Fatalf("Register(%q) = %v", p.String(), err)
				}
			}
			ValidateClient(t, c, tt.tests)
		})
	}
}

type StaticIDP struct {
	name   string
	wantId string
	human  string
}

var _ IdentityProvider = (*StaticIDP)(nil)

// Resolve implements IdentityProvider.
func (s *StaticIDP) Resolve(_ context.Context, id string) (*Identity, error) {
	if id == s.wantId {
		return &Identity{
			UserID:    id,
			HumanName: s.human,
			Provider:  s,
		}, nil
	}
	return nil, errors.New("not found")
}

// String implements IdentityProvider.
func (s *StaticIDP) String() string {
	return s.name
}

// URL implements IdentityProvider.
func (s *StaticIDP) URL() url.URL {
	return url.URL{
		Scheme: "https",
		Host:   s.name,
		Path:   "/token",
	}
}

// Validate implements IdentityProvider.
func (s *StaticIDP) Validate(_ context.Context, token jwt.Token) (*Identity, error) {
	sURL := s.URL()
	if token.Subject() == s.wantId && token.Issuer() == sURL.String() {
		return &Identity{
			UserID:    token.Subject(),
			HumanName: s.human,
			Provider:  s,
		}, nil
	}
	return nil, errors.New("not found")
}
