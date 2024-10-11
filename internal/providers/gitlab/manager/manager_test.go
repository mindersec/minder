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
