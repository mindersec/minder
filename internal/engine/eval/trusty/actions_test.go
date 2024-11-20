// SPDX-FileCopyrightText: Copyright 2023 The Minder Authors
// SPDX-License-Identifier: Apache-2.0

// Package trusty provides an evaluator that uses the trusty API
package trusty

import (
	"testing"

	"github.com/stretchr/testify/require"

	v1 "github.com/mindersec/minder/pkg/api/protobuf/go/minder/v1"
)

func TestNewSummaryPrHandler(t *testing.T) {
	t.Parallel()

	// newSummaryPrHandler must never fail. The only failure point
	// right now is the pr comment template
	_, err := newSummaryPrHandler(&v1.PullRequest{}, nil, "")
	require.NoError(t, err)
}

func TestBuildProvenanceStruct(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		sut      *trustyReport
		mustNil  bool
		expected *templateProvenance
	}{
		{
			name: "full-response",
			sut: &trustyReport{
				Provenance: &provenance{
					Historical: &historicalProvenance{
						Tags:     10,
						Common:   8,
						Overlap:  80,
						Versions: 10,
					},
					Sigstore: &sigstoreProvenance{
						Issuer:           "CN=sigstore-intermediate,O=sigstore.dev",
						Workflow:         ".github/workflows/build_and_deploy.yml",
						SourceRepository: "https://github.com/vercel/next.js",
						RekorURI:         "https://search.sigstore.dev/?logIndex=88381843",
					},
				},
			},
			mustNil: false,
			expected: &templateProvenance{
				Historical: &templateHistoricalProvenance{
					NumVersions:     10,
					NumTags:         10,
					MatchedVersions: 8,
				},
				Sigstore: &templateSigstoreProvenance{
					SourceRepository: "https://github.com/vercel/next.js",
					Workflow:         ".github/workflows/build_and_deploy.yml",
					Issuer:           "CN=sigstore-intermediate,O=sigstore.dev",
					RekorURI:         "https://search.sigstore.dev/?logIndex=88381843",
				},
			},
		},
		{
			name: "only-historical",
			sut: &trustyReport{
				Provenance: &provenance{
					Historical: &historicalProvenance{
						Tags:     10,
						Common:   8,
						Overlap:  80,
						Versions: 10,
					},
				},
			},
			mustNil: false,
			expected: &templateProvenance{
				Historical: &templateHistoricalProvenance{
					NumVersions:     10,
					NumTags:         10,
					MatchedVersions: 8,
				},
			},
		},
		{
			name: "only-sigstore",
			sut: &trustyReport{
				Provenance: &provenance{
					Sigstore: &sigstoreProvenance{
						Issuer:           "CN=sigstore-intermediate,O=sigstore.dev",
						Workflow:         ".github/workflows/build_and_deploy.yml",
						SourceRepository: "https://github.com/vercel/next.js",
						RekorURI:         "https://search.sigstore.dev/?logIndex=88381843",
					},
				},
			},
			mustNil: false,
			expected: &templateProvenance{
				Sigstore: &templateSigstoreProvenance{
					SourceRepository: "https://github.com/vercel/next.js",
					Workflow:         ".github/workflows/build_and_deploy.yml",
					Issuer:           "CN=sigstore-intermediate,O=sigstore.dev",
					RekorURI:         "https://search.sigstore.dev/?logIndex=88381843",
				},
			},
		},
		{
			name:    "no-response",
			sut:     nil,
			mustNil: true,
		},
		{
			name:    "no-provenance",
			sut:     &trustyReport{},
			mustNil: true,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res := buildProvenanceStruct(tc.sut)
			if tc.mustNil {
				require.Nil(t, res)
				return
			}

			if tc.expected.Historical == nil {
				require.Nil(t, res.Historical)
			} else {
				require.Equal(t, tc.expected.Historical.MatchedVersions, res.Historical.MatchedVersions)
				require.Equal(t, tc.expected.Historical.NumTags, res.Historical.NumTags)
				require.Equal(t, tc.expected.Historical.NumVersions, res.Historical.NumVersions)
			}

			if tc.expected.Sigstore == nil {
				require.Nil(t, res.Sigstore)
			} else {
				require.Equal(t, tc.expected.Sigstore.Issuer, res.Sigstore.Issuer)
				require.Equal(t, tc.expected.Sigstore.Workflow, res.Sigstore.Workflow)
				require.Equal(t, tc.expected.Sigstore.RekorURI, res.Sigstore.RekorURI)
				require.Equal(t, tc.expected.Sigstore.SourceRepository, res.Sigstore.SourceRepository)
			}
		})
	}
}
