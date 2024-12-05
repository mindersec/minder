// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package manager contains the GitLabProviderClassManager
package manager

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func Test_tokenNeedsRefresh(t *testing.T) {
	t.Parallel()

	baseTime := time.Now()

	tests := []struct {
		name  string
		token oauth2.Token
		want  bool
	}{
		{
			name:  "token is expired",
			token: accessTokenWithExpiration(baseTime.Add(-1 * time.Minute)),
			want:  true,
		},
		{
			name:  "token is not expired and does not need refresh",
			token: accessTokenWithExpiration(baseTime.Add(15 * time.Minute)),
			want:  false,
		},
		{
			name:  "token is not expired but needs refresh",
			token: accessTokenWithExpiration(baseTime.Add(5 * time.Minute)),
			want:  true,
		},
		{
			name: "token is not valid",
			token: oauth2.Token{
				AccessToken: "",
				Expiry:      baseTime.Add(15 * time.Minute),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			needsRefresh := tokenNeedsRefresh(tt.token)
			assert.Equal(t, tt.want, needsRefresh)
		})
	}
}

func accessTokenWithExpiration(exp time.Time) oauth2.Token {
	return oauth2.Token{
		AccessToken: "ozz-likes-beer",
		Expiry:      exp,
	}
}
