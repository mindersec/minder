// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCurlCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		method      string
		apiBaseURL  string
		endpoint    string
		body        string
		wantError   bool
		wantString  string
	}{
		{
			name:       "valid curl command standard",
			method:     "POST",
			apiBaseURL: "https://api.github.com",
			endpoint:   "/repos/mindersec/minder/issues",
			body:       `{"title":"test"}`,
			wantError:  false,
			wantString: `curl -L -X POST \
 -H "Accept: application/vnd.github+json" \
 -H "Authorization: Bearer $TOKEN" \
 -H "X-GitHub-Api-Version: 2022-11-28" \
 https://api.github.com/repos/mindersec/minder/issues \
 -d '{"title":"test"}'`,
		},
		{
			name:       "escape single quotes",
			method:     "POST",
			apiBaseURL: "https://api.github.com",
			endpoint:   "/repos/mindersec/minder/issues",
			body:       `{"text": "can't do this; rm -rf *"}`,
			wantError:  false,
			wantString: `curl -L -X POST \
 -H "Accept: application/vnd.github+json" \
 -H "Authorization: Bearer $TOKEN" \
 -H "X-GitHub-Api-Version: 2022-11-28" \
 https://api.github.com/repos/mindersec/minder/issues \
 -d '{"text": "can'\''t do this; rm -rf *"}'`,
		},
		{
			name:       "empty method",
			method:     "",
			apiBaseURL: "https://api.github.com",
			wantError:  true,
		},
		{
			name:       "empty api base url",
			method:     "GET",
			apiBaseURL: "",
			wantError:  true,
		},
		{
			name:       "invalid url format",
			method:     "GET",
			apiBaseURL: "http://[::1]:namedport", // invalid port
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := GenerateCurlCommand(context.Background(), tt.method, tt.apiBaseURL, tt.endpoint, tt.body)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			assert.Equal(t, tt.wantString, got)
		})
	}
}
