// SPDX-FileCopyrightText: Copyright 2024 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package controlplane

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetRemediationURLFromMetadata(t *testing.T) {
	validData := []byte(`{"pr_number": 18}`)
	t.Parallel()
	for _, tc := range []struct {
		name        string
		data        []byte
		repo        string
		expectedURL string
		mustErr     bool
	}{
		{"normal", validData, "My-Example_1.0/Test_2", "https://github.com/My-Example_1.0/Test_2/pull/18", false},
		{"invalid-slug", validData, "example", "", true},
		{"no-pr", []byte(`{}`), "example/test", "", false},
		{"no-slug", validData, "", "", true},
		{"no-slug-no-pr", []byte(`{}`), "", "", true},
		{"invalid-json", []byte(`Yo!`), "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			url, err := getRemediationURLFromMetadata(tc.data, tc.repo)
			if tc.mustErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedURL, url)
		})
	}

}

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
		{"normal", validPayload, "example/test", "https://github.com/example/test/security/advisories/GHAS-advisory_ID_here", false},
		{"no-repo", validPayload, "", "", true},
		{"bad-json", []byte(`invalid _`), "", "", true},
		{"no-advisory", []byte(`{"ghsa_id": ""}`), "", "", true},
		{"invalid-slug", []byte(`{"ghsa_id": "abc"}`), "invalid slug", "", true},
	} {
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
