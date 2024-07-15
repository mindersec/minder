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

package vulncheck

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pbinternal "github.com/stacklok/minder/internal/proto"
)

const multipleRanges = `
{
  "vulns": [
    {
      "id": "GHSA-123",
      "summary": "Summary",
      "details": "Details",
      "aliases": [
        "CVE-2023-39347"
      ],
      "modified": "2023-09-27T15:42:11Z",
      "published": "2023-09-26T18:00:22Z",
      "database_specific": {},
      "references": [],
      "affected": [
        {
          "package": {
            "name": "golang.org/x/text",
            "ecosystem": "Go",
            "purl": "pkg:golang/golang.org/x/text"
          },
          "ranges": [
            {
              "type": "SEMVER",
              "events": [
                {
                  "introduced": "1.13.0"
                },
                {
                  "fixed": "1.13.7"
                }
              ]
            }
          ],
          "database_specific": {
            "source": "Source",
            "last_known_affected_version_range": "<= 1.13.6"
          }
        },
        {
          "package": {
            "name": "golang.org/x/text",
            "ecosystem": "Go",
            "purl": "pkg:golang/golang.org/x/text"
          },
          "ranges": [
            {
              "type": "SEMVER",
              "events": [
                {
                  "introduced": "1.14.0"
                },
                {
                  "fixed": "1.14.2"
                }
              ]
            }
          ],
          "database_specific": {
            "source": "Source",
            "last_known_affected_version_range": "<= 1.14.1"
          }
        },
        {
          "package": {
            "name": "golang.org/x/text",
            "ecosystem": "Go",
            "purl": "pkg:golang/golang.org/x/text"
          },
          "ranges": [
            {
              "type": "SEMVER",
              "events": [
                {
                  "introduced": "0"
                },
                {
                  "fixed": "1.12.14"
                }
              ]
            }
          ],
          "database_specific": {
            "source": "Source",
            "last_known_affected_version_range": "<= 1.12.13"
          }
        }
      ],
      "schema_version": "1.6.0",
      "severity": []
    }
  ]
}
`

const nonSemver = `
{
  "vulns": [
    {
      "id": "GHSA-123",
      "summary": "Summary",
      "details": "Details",
      "aliases": [
        "CVE-2023-39347"
      ],
      "modified": "2023-09-27T15:42:11Z",
      "published": "2023-09-26T18:00:22Z",
      "database_specific": {},
      "references": [],
      "affected": [
        {
          "package": {
            "name": "golang.org/x/text",
            "ecosystem": "Go",
            "purl": "pkg:golang/golang.org/x/text"
          },
          "ranges": [
            {
              "type": "GIT",
              "events": [
                {
                  "introduced": "commitHash1"
                },
                {
                  "fixed": "commitHash2"
                }
              ]
            }
          ],
          "database_specific": {
            "source": "Source"
          }
        }
      ],
      "schema_version": "1.6.0",
      "severity": []
    }
  ]
}
`

const notFixed = `
{
  "vulns": [
    {
      "id": "GHSA-123",
      "summary": "Summary",
      "details": "Details",
      "aliases": [
        "CVE-2023-39347"
      ],
      "modified": "2023-09-27T15:42:11Z",
      "published": "2023-09-26T18:00:22Z",
      "database_specific": {},
      "references": [],
      "affected": [
        {
          "package": {
            "name": "golang.org/x/text",
            "ecosystem": "Go",
            "purl": "pkg:golang/golang.org/x/text"
          },
          "ranges": [
            {
              "type": "SEMVER",
              "events": [
                {
                  "introduced": "0"
                }
              ]
            }
          ],
          "database_specific": {
            "source": "Source"
          }
        }
      ],
      "schema_version": "1.6.0",
      "severity": []
    }
  ]
}
`

func TestGoVulnDb(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		mockVulnHandler http.HandlerFunc
		depName         string
		depVersion      string
		expectError     bool
		expectReply     *VulnerabilityResponse
	}{
		{
			name: "SemverMultipleRanges",
			mockVulnHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(multipleRanges))
				if err != nil {
					t.Fatal(err)
				}
			},
			depName:    "golang.org/x/text",
			depVersion: "v1.13.1",
			expectReply: &VulnerabilityResponse{
				Vulns: []Vulnerability{
					{
						ID:         "GHSA-123",
						Summary:    "Summary",
						Details:    "Details",
						Introduced: "1.13.0",
						Fixed:      "1.13.7",
						Type:       "SEMVER",
					},
				},
			},
		},
		{
			name: "NonSemver",
			mockVulnHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(nonSemver))
				if err != nil {
					t.Fatal(err)
				}
			},
			depName:    "golang.org/x/text",
			depVersion: "v1.13.1",
			expectReply: &VulnerabilityResponse{
				Vulns: []Vulnerability{
					{
						ID:         "GHSA-123",
						Summary:    "Summary",
						Details:    "Details",
						Introduced: "commitHash1",
						Fixed:      "commitHash2",
						Type:       "GIT",
					},
				},
			},
		},
		{
			name: "NotFixed",
			mockVulnHandler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(notFixed))
				if err != nil {
					t.Fatal(err)
				}
			},
			depName:    "golang.org/x/text",
			depVersion: "v1.13.1",
			expectReply: &VulnerabilityResponse{
				Vulns: []Vulnerability{
					{
						ID:         "GHSA-123",
						Summary:    "Summary",
						Details:    "Details",
						Introduced: "0",
						Fixed:      "",
						Type:       "SEMVER",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vulnServer := httptest.NewServer(tt.mockVulnHandler)
			defer vulnServer.Close()

			db := newOsvDb(vulnServer.URL)
			assert.NotNil(t, db, "Failed to create OSV DB")

			dep := &pbinternal.Dependency{
				Name:    tt.depName,
				Version: tt.depVersion,
			}

			r, err := http.NewRequest("POST", vulnServer.URL, bytes.NewReader([]byte(`{}`)))
			require.NoError(t, err, "failed to create request")

			reply, err := db.SendRecvRequest(r, dep)
			if tt.expectError {
				assert.Error(t, err, "Expected error")
			} else {
				assert.NoError(t, err, "Expected no error")
				require.Equal(t, tt.expectReply, reply, "expected reply to match mock data")
			}
		})
	}
}
