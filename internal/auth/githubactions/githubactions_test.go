// SPDX-FileCopyrightText: Copyright 2025 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package githubactions provides an implementation of the GitHub IdentityProvider.
package githubactions

import (
	"context"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwt"

	"github.com/mindersec/minder/internal/auth"
)

func TestGitHubActions_Resolve(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		identity string
		want     *auth.Identity
	}{{
		name:     "Resolve from storage",
		identity: "repo+evankanderson/actions-id-token-testing+ref+refs/heads/main",
		want: &auth.Identity{
			HumanName: "repo:evankanderson/actions-id-token-testing:ref:refs/heads/main",
			UserID:    "repo+evankanderson/actions-id-token-testing+ref+refs/heads/main",
		},
	}, {
		name:     "Resolve from human input",
		identity: "repo:evankanderson/actions-id-token-testing:ref:refs/heads/main",
		want: &auth.Identity{
			HumanName: "repo:evankanderson/actions-id-token-testing:ref:refs/heads/main",
			UserID:    "repo+evankanderson/actions-id-token-testing+ref+refs/heads/main",
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gha := &GitHubActions{}

			got, err := gha.Resolve(context.Background(), tt.identity)
			if err != nil {
				t.Errorf("GitHubActions.Resolve() error = %v", err)
			}

			tt.want.Provider = gha
			if tt.want.String() != got.String() {
				t.Errorf("GitHubActions.Resolve() = %v, want %v", got.String(), tt.want.String())
			}
			if tt.want.Human() != got.Human() {
				t.Errorf("GitHubActions.Resolve() = %v, want %v", got.Human(), tt.want.Human())
			}
		})
	}
}

func TestGitHubActions_Validate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   func() jwt.Token
		want    *auth.Identity
		wantErr bool
	}{{
		name: "Validate token",
		input: func() jwt.Token {
			tok := jwt.New()
			_ = tok.Set("iss", "https://token.actions.githubusercontent.com")
			_ = tok.Set("sub", "repo:evankanderson/actions-id-token-testing:ref:refs/heads/main")
			return tok
		},
		want: &auth.Identity{
			HumanName: "repo:evankanderson/actions-id-token-testing:ref:refs/heads/main",
			UserID:    "repo+evankanderson/actions-id-token-testing+ref+refs/heads/main",
		},
	}, {
		name: "Validate token with invalid issuer",
		input: func() jwt.Token {
			tok := jwt.New()
			_ = tok.Set("iss", "https://issuer.minder.com/")
			_ = tok.Set("sub", "repo:evankanderson/actions-id-token-testing:ref:refs/heads/main")
			return tok
		},
		want:    nil,
		wantErr: true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gha := &GitHubActions{}
			got, err := gha.Validate(context.Background(), tt.input())
			if (err != nil) != tt.wantErr {
				t.Errorf("GitHubActions.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				tt.want.Provider = gha
			}
			if tt.want.String() != got.String() {
				t.Errorf("GitHubActions.Validate() = %v, want %v", got.String(), tt.want.String())
			}
			if tt.want.Human() != got.Human() {
				t.Errorf("GitHubActions.Validate() = %v, want %v", got.Human(), tt.want.Human())
			}
		})
	}
}
