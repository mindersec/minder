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
			name:      "Invalid json",
			input:     `<!-- minder: pr-status-body: { "ContentSha": "abcdef", "ReviewID": 100 } -->`, // ReviewID is not string block in json
			wantError: true,
		},
		{
			name:      "No match",
			input:     `some random text`,
			wantError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
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

func TestGenerateMetadata(t *testing.T) {
	t.Parallel()

	s := &statusReport{
		CommitSHA: "abcdef",
		TrackedDependencies: []dependencyVulnerabilities{
			{
				Vulnerabilities: []Vulnerability{{ID: "1"}, {ID: "2"}},
				PatchVersion:    "1.2.3",
			},
			{
				Vulnerabilities: []Vulnerability{},
			},
			{
				Vulnerabilities: []Vulnerability{{ID: "3"}},
			},
		},
	}

	meta := s.generateMetadata()
	assert.Equal(t, 2, meta.VulnerabilityCount)
	assert.Equal(t, 1, meta.RemediationCount)
	assert.Equal(t, 3, meta.TrackedDepsCount)
	assert.Equal(t, "abcdef", meta.CommitSHA)
}

func TestVulnSummaryReportRender(t *testing.T) {
	t.Parallel()

	t.Run("Empty Dependencies", func(t *testing.T) {
		r := &vulnSummaryReport{}
		out, err := r.render()
		require.NoError(t, err)
		assert.Contains(t, out, "analyzed this PR and found it does not add any new vulnerable dependencies")
	})

	t.Run("With Dependencies", func(t *testing.T) {
		r := &vulnSummaryReport{
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
				},
			},
		}
		out, err := r.render()
		require.NoError(t, err)
		assert.Contains(t, out, "Summary of vulnerabilities found")
		assert.Contains(t, out, "test-pkg")
		assert.Contains(t, out, "1.0.0")
		assert.Contains(t, out, "CVE-2023-1234")
		assert.Contains(t, out, "A test vulnerability")
	})
}

func TestStatusReportRender(t *testing.T) {
	t.Parallel()

	t.Run("No Dependencies", func(t *testing.T) {
		s := &statusReport{
			StatusText: "No vulns found",
			CommitSHA:  "1234567890",
			ReviewID:   42,
		}
		out, err := s.render()
		require.NoError(t, err)
		assert.Contains(t, out, "No vulns found")
		assert.Contains(t, out, "12345678")
		assert.Contains(t, out, "<!-- minder: pr-status-body:")
	})

	t.Run("With Dependencies", func(t *testing.T) {
		s := &statusReport{
			StatusText: "Vulns found!",
			CommitSHA:  "1234567890",
			ReviewID:   42,
			TrackedDependencies: []dependencyVulnerabilities{
				{
					Dependency: &pbinternal.Dependency{
						Name:    "bad-pkg",
						Version: "1.0.0",
					},
					Vulnerabilities: []Vulnerability{{ID: "1"}},
					PatchVersion:    "1.0.1",
				},
			},
		}
		out, err := s.render()
		require.NoError(t, err)
		assert.Contains(t, out, "Vulns found!")
		assert.Contains(t, out, "bad-pkg")
		assert.Contains(t, out, "1.0.1")
		assert.Contains(t, out, "Summary of vulnerabilities found") // vulnSummary is appended
	})
}
