// Copyright 2024 Stacklok, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAlertURLFromMetadata(t *testing.T) {
	t.Parallel()
	validPayload := []byte(`{"ghsa_id": "GHAS-advisory_ID_here"}`)
	for _, tc := range []struct {
		name     string
		data     []byte
		repo     string
		expected string
		mustErr  bool
	}{
		{
			name: "normal", data: validPayload, repo: "example/test",
			expected: "https://github.com/example/test/security/advisories/GHAS-advisory_ID_here", mustErr: false,
		},
		{name: "no-repo", data: validPayload, repo: "", expected: "", mustErr: true},
		{name: "bad-json", data: []byte(`invalid _`), repo: "", expected: "", mustErr: true},
		{name: "no-advisory", data: []byte(`{"ghsa_id": ""}`), repo: "", expected: "", mustErr: true},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res, err := getAlertURLFromMetadata(tc.data, tc.repo)
			if tc.mustErr {
				require.Error(t, err)
				return
			}
			require.Equal(t, tc.expected, res)
		})
	}
}
