// Copyright 2023 Stacklok, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package trusty provides an evaluator that uses the trusty API
package trusty

import (
	"testing"

	trustytypes "github.com/stacklok/trusty-sdk-go/pkg/types"
	"github.com/stretchr/testify/require"

	v1 "github.com/stacklok/minder/pkg/api/protobuf/go/minder/v1"
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
		sut      *trustytypes.Reply
		mustNil  bool
		expected *templateProvenance
	}{
		{
			name: "full-response",
			sut: &trustytypes.Reply{
				Provenance: &trustytypes.Provenance{
					Score: 8.0,
					Description: trustytypes.ProvenanceDescription{
						Historical: trustytypes.HistoricalProvenance{
							Tags:     10,
							Common:   8,
							Overlap:  80,
							Versions: 10,
						},
						Sigstore: trustytypes.SigstoreProvenance{
							Issuer:           "CN=sigstore-intermediate,O=sigstore.dev",
							Workflow:         ".github/workflows/build_and_deploy.yml",
							SourceRepository: "https://github.com/vercel/next.js",
							Transparency:     "https://search.sigstore.dev/?logIndex=88381843",
						},
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
			sut: &trustytypes.Reply{
				Provenance: &trustytypes.Provenance{
					Score: 8.0,
					Description: trustytypes.ProvenanceDescription{
						Historical: trustytypes.HistoricalProvenance{
							Tags:     10,
							Common:   8,
							Overlap:  80,
							Versions: 10,
						},
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
			sut: &trustytypes.Reply{
				Provenance: &trustytypes.Provenance{
					Score: 8.0,
					Description: trustytypes.ProvenanceDescription{
						Sigstore: trustytypes.SigstoreProvenance{
							Issuer:           "CN=sigstore-intermediate,O=sigstore.dev",
							Workflow:         ".github/workflows/build_and_deploy.yml",
							SourceRepository: "https://github.com/vercel/next.js",
							Transparency:     "https://search.sigstore.dev/?logIndex=88381843",
						},
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
			sut:     &trustytypes.Reply{},
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
