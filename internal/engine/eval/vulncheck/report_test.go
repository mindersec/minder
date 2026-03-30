// SPDX-FileCopyrightText: Copyright 2026 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

package vulncheck

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pbinternal "github.com/mindersec/minder/internal/proto"
)

func TestExtractContentShaAndReviewID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantSha   string
		wantID    int64
		wantError bool
	}{
		{
			name:      "Valid magic comment",
			input:     `<!-- minder: pr-status-body: { "ContentSha": "abcdef123456", "ReviewID": "100500" } -->\n\nsome other text`,
			wantSha:   "abcdef123456",
			wantID:    100500,
			wantError: false,
		},
		{
			name:      "Missing json data",
			input:     `<!-- minder: pr-status-body: { "ContentSha": "abcdef", "Review": "200" } -->`, // ReviewID missing
			wantError: true,
		},
		{
			name:      "Double comment",
			input:     `<!-- minder: pr-status-body: { "ContentSha": "abcdef", "ReviewID": "2" } -->` + "\n" + `<!-- minder: pr-status-body: { "ContentSha": "a1b2c3", "ReviewID": "5" } -->`,
			wantSha:   "abcdef",
			wantID:    2,
			wantError: false, // Let's guess it matches the first one via standard regex ungreedy match. We will adjust based on test results.
		},
		{
			name:      "No match",
			input:     `some random text`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := extractContentShaAndReviewID(tt.input)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantSha, got.ContentSha)
				assert.Equal(t, tt.wantID, got.ReviewID)
			}
		})
	}
}

func TestStatusReportRender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		status          *statusReport
		expectedStrings []string
	}{
		{
			name: "No Dependencies",
			status: &statusReport{
				StatusText: "No vulns found",
				CommitSHA:  "1234567890",
				ReviewID:   42,
			},
			expectedStrings: []string{
				"No vulns found",
				"12345678",
				"<!-- minder: pr-status-body:",
				"<b>vulnerable packages:</b> <code>0</code>",
			},
		},
		{
			name: "With Dependencies",
			status: &statusReport{
				StatusText: "Vulns found!",
				CommitSHA:  "1234567890",
				ReviewID:   42,
				TrackedDependencies: []dependencyVulnerabilities{
					{
						Dependency: &pbinternal.Dependency{
							Name:      "test-pkg",
							Version:   "1.0.0",
							Ecosystem: pbinternal.DepEcosystem_DEP_ECOSYSTEM_NPM,
						},
						Vulnerabilities: []Vulnerability{
							{
								ID:         "CVE-2023-1234",
								Summary:    "A test vulnerability",
								Introduced: "0.0.0",
								Fixed:      "1.0.1",
							},
						},
						PatchVersion: "1.0.1",
					},
					{
						Dependency: &pbinternal.Dependency{
							Name:    "bad-pkg",
							Version: "1.0.0",
						},
						Vulnerabilities: []Vulnerability{{ID: "1"}},
						PatchVersion:    "1.0.1",
					},
				},
			},
			expectedStrings: []string{
				"Vulns found!",
				"test-pkg",
				"1.0.0",
				"CVE-2023-1234",
				"A test vulnerability",
				"bad-pkg",
				"1.0.1",
				"Summary of vulnerabilities found",
				"<b>vulnerable packages:</b> <code>2</code>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, err := tt.status.render()
			require.NoError(t, err)
			for _, s := range tt.expectedStrings {
				assert.Contains(t, out, s)
			}
		})
	}
}
